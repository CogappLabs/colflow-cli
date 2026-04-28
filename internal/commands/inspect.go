package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

func NewInspect() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [file.parquet | asset_name]",
		Short: "Inspect a parquet file: schema, row count, null %, file size",
		Long:  "Inspect a parquet file. If no path given, lists output/ to pick. Bare names resolve to <project>/output/<name>.parquet.",
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
				return fmt.Errorf("open parquet: %w", err)
			}

			fmt.Println(format.Bold(path))
			fmt.Println()
			fmt.Printf("  %-12s %s\n", "Size:", format.Cyan(humanBytes(fi.Size())))
			fmt.Printf("  %-12s %s\n", "Rows:", format.Cyan(fmt.Sprintf("%d", pf.NumRows())))
			fmt.Printf("  %-12s %s\n", "Row groups:", fmt.Sprintf("%d", len(pf.RowGroups())))
			fmt.Println()

			schema := pf.Schema()
			cols := schema.Columns()
			fmt.Println(format.Bold("Schema:"))

			leafTypes := map[string]string{}
			leafOptional := map[string]bool{}
			for _, path := range cols {
				var node parquet.Node = schema
				optional := false
				for _, p := range path {
					next, ok := nodeChild(node, p)
					if !ok {
						break
					}
					if next.Optional() {
						optional = true
					}
					node = next
				}
				key := strings.Join(path, ".")
				leafTypes[key] = nodeTypeName(node)
				leafOptional[key] = optional
			}

			maxName := 0
			for _, path := range cols {
				k := strings.Join(path, ".")
				if len(k) > maxName {
					maxName = len(k)
				}
			}

			for _, path := range cols {
				k := strings.Join(path, ".")
				nullable := "required"
				if leafOptional[k] {
					nullable = format.Gray("optional")
				} else {
					nullable = format.Yellow(nullable)
				}
				fmt.Printf("  %s  %s  %s\n",
					format.PadRight(k, maxName),
					format.PadRight(format.Gray(leafTypes[k]), 24),
					nullable,
				)
			}

			fmt.Println()
			fmt.Println(format.Bold("Null counts:"))
			nulls := computeNulls(pf, cols)
			total := pf.NumRows()
			for _, path := range cols {
				k := strings.Join(path, ".")
				n := nulls[k]
				pct := 0.0
				if total > 0 {
					pct = 100.0 * float64(n) / float64(total)
				}
				colour := format.Green
				if pct > 0 {
					colour = format.Yellow
				}
				if pct > 50 {
					colour = format.Red
				}
				fmt.Printf("  %s  %s  %s\n",
					format.PadRight(k, maxName),
					format.PadRight(fmt.Sprintf("%d", n), 12),
					colour(fmt.Sprintf("%.1f%%", pct)),
				)
			}
			return nil
		},
	}
	return cmd
}

func resolveOrPick(args []string) (string, error) {
	if len(args) == 1 {
		return project.ResolveParquet(args[0])
	}
	files, err := project.ListParquets()
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no parquet files in output/")
	}
	items := make([]prompts.Item[string], len(files))
	for i, f := range files {
		items[i] = prompts.Item[string]{Display: filepath.Base(f), Value: f}
	}
	picked, ok := prompts.Pick("Parquet files in output/", items)
	if !ok {
		return "", nil
	}
	return picked, nil
}

func nodeChild(n parquet.Node, name string) (parquet.Node, bool) {
	for _, f := range n.Fields() {
		if f.Name() == name {
			return f, true
		}
	}
	return nil, false
}

func nodeTypeName(n parquet.Node) string {
	if n == nil || !n.Leaf() {
		return "group"
	}
	t := n.Type()
	if t == nil {
		return "?"
	}
	if lt := t.LogicalType(); lt != nil {
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", lt)))
	}
	return strings.ToLower(t.Kind().String())
}

func computeNulls(pf *parquet.File, cols [][]string) map[string]int64 {
	out := map[string]int64{}
	for _, path := range cols {
		key := strings.Join(path, ".")
		var nulls int64
		for _, rg := range pf.RowGroups() {
			leaf, ok := rg.Schema().Lookup(path...)
			if !ok {
				continue
			}
			cc := rg.ColumnChunks()[leaf.ColumnIndex]
			pages := cc.Pages()
			for {
				page, err := pages.ReadPage()
				if err != nil || page == nil {
					break
				}
				nulls += page.NumNulls()
				parquet.Release(page)
			}
			pages.Close()
		}
		out[key] = nulls
	}
	return out
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
