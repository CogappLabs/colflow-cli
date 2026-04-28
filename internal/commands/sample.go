package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

func NewSample() *cobra.Command {
	var n int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "sample [file.parquet | asset_name]",
		Short: "Pretty-print N rows from a parquet file",
		Long:  "Pretty-print rows. If no path given, lists output/ to pick. Bare names resolve to <project>/output/<name>.parquet.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveOrPick(args)
			if err != nil {
				return err
			}
			if path == "" {
				return nil
			}
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("file not found: %s", path)
			}
			if !strings.HasSuffix(strings.ToLower(path), ".parquet") {
				return fmt.Errorf("expected .parquet file, got: %s", path)
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			pf, err := parquet.OpenFile(f, fi.Size())
			if err != nil {
				return err
			}

			schema := pf.Schema()
			reader := parquet.NewReader(pf, schema)
			defer reader.Close()

			rows := make([]parquet.Row, n)
			read, err := reader.ReadRows(rows)
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			rows = rows[:read]

			cols := schema.Columns()

			if asJSON {
				out := make([]map[string]any, len(rows))
				for i, row := range rows {
					out[i] = rowToMap(schema, cols, row)
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			if len(rows) == 0 {
				fmt.Println(format.Gray("(no rows)"))
				return nil
			}

			colNames := make([]string, len(cols))
			maxName := 0
			for i, c := range cols {
				colNames[i] = strings.Join(c, ".")
				if len(colNames[i]) > maxName {
					maxName = len(colNames[i])
				}
			}

			for i, row := range rows {
				fmt.Println(format.Bold(fmt.Sprintf("Row %d:", i+1)))
				m := rowToMap(schema, cols, row)
				for _, name := range colNames {
					val := lookupNested(m, strings.Split(name, "."))
					fmt.Printf("  %s  %s\n", format.PadRight(format.Cyan(name), maxName), formatValue(val))
				}
				fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&n, "rows", "n", 5, "Number of rows")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

func rowToMap(schema *parquet.Schema, cols [][]string, row parquet.Row) map[string]any {
	out := map[string]any{}
	leafByIdx := map[int][]string{}
	for _, path := range cols {
		leaf, ok := schema.Lookup(path...)
		if !ok {
			continue
		}
		leafByIdx[leaf.ColumnIndex] = path
	}
	for _, v := range row {
		path, ok := leafByIdx[v.Column()]
		if !ok {
			continue
		}
		setNested(out, path, valueOf(v))
	}
	return out
}

func setNested(m map[string]any, path []string, val any) {
	for i, p := range path {
		if i == len(path)-1 {
			m[p] = val
			return
		}
		next, ok := m[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[p] = next
		}
		m = next
	}
}

func valueOf(v parquet.Value) any {
	if v.IsNull() {
		return nil
	}
	switch v.Kind() {
	case parquet.Boolean:
		return v.Boolean()
	case parquet.Int32:
		return v.Int32()
	case parquet.Int64:
		return v.Int64()
	case parquet.Float:
		return v.Float()
	case parquet.Double:
		return v.Double()
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return string(v.ByteArray())
	default:
		return v.String()
	}
}

func lookupNested(m map[string]any, path []string) any {
	if len(path) == 0 {
		return nil
	}
	v, ok := m[path[0]]
	if !ok {
		return nil
	}
	if len(path) == 1 {
		return v
	}
	if next, ok := v.(map[string]any); ok {
		return lookupNested(next, path[1:])
	}
	return v
}

func formatValue(v any) string {
	if v == nil {
		return format.Gray("null")
	}
	switch x := v.(type) {
	case string:
		if len(x) > 80 {
			return fmt.Sprintf("%q…", x[:80])
		}
		return fmt.Sprintf("%q", x)
	case []byte:
		s := string(x)
		if len(s) > 80 {
			return fmt.Sprintf("%q…", s[:80])
		}
		return fmt.Sprintf("%q", s)
	case map[string]any:
		b, _ := json.Marshal(x)
		return string(b)
	case []any:
		b, _ := json.Marshal(x)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}
