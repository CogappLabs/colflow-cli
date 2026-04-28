package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/parquet-go/parquet-go"
	"github.com/urfave/cli/v3"
)

func NewSample() *cli.Command {
	return &cli.Command{
		Name:        "sample",
		Usage:       "Pretty-print N rows from a parquet file (optionally filtered)",
		Description: "Pretty-print rows. If no path given, lists output/ to pick. Bare names resolve to <project>/output/<name>.parquet. Repeatable --where field=value filters by equality (dot-paths supported, e.g. artist.name=Alice).",
		ArgsUsage:   "[file.parquet | asset_name]",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "rows", Aliases: []string{"n"}, Value: 5, Usage: "Number of rows to return"},
			&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
			&cli.StringSliceFlag{Name: "where", Usage: "Filter rows by field=value (repeatable, dot-paths for nested)"},
			&cli.IntFlag{Name: "max-scan", Value: 1_000_000, Usage: "Max rows scanned when filtering before giving up"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() > 1 {
				return fmt.Errorf("sample takes at most 1 argument")
			}
			args := c.Args().Slice()
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

			n := int(c.Int("rows"))
			maxScan := int(c.Int("max-scan"))
			asJSON := c.Bool("json")
			where := c.StringSlice("where")

			filters, err := parseWhere(where)
			if err != nil {
				return err
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

			cols := schema.Columns()
			rows := []parquet.Row{}
			scanned := 0
			batchSize := 64
			if len(filters) == 0 {
				batchSize = n
			}
			for len(rows) < n && scanned < maxScan {
				batch := make([]parquet.Row, batchSize)
				read, err := reader.ReadRows(batch)
				if read > 0 {
					for i := 0; i < read && len(rows) < n; i++ {
						scanned++
						if len(filters) == 0 || matchesFilters(rowToMap(schema, cols, batch[i]), filters) {
							rows = append(rows, batch[i])
						}
					}
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return err
				}
			}

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
			for i, cc := range cols {
				colNames[i] = strings.Join(cc, ".")
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
}

type filter struct {
	Path  []string
	Value string
}

func parseWhere(specs []string) ([]filter, error) {
	out := make([]filter, 0, len(specs))
	for _, s := range specs {
		idx := strings.Index(s, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("--where %q must be field=value", s)
		}
		field := strings.TrimSpace(s[:idx])
		val := s[idx+1:]
		out = append(out, filter{Path: strings.Split(field, "."), Value: val})
	}
	return out, nil
}

func matchesFilters(row map[string]any, filters []filter) bool {
	for _, f := range filters {
		v := lookupNested(row, f.Path)
		if v == nil {
			if f.Value == "" || strings.EqualFold(f.Value, "null") {
				continue
			}
			return false
		}
		if fmt.Sprint(v) != f.Value {
			return false
		}
	}
	return true
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
