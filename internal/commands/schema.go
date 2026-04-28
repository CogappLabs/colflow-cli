package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

func NewSchema() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "schema [file.parquet | asset_name]",
		Short: "List the schema of a parquet file",
		Long:  "Show columns + types for a parquet file. Bare names resolve to <project>/output/<name>.parquet.",
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
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			pf, err := parquet.OpenFile(f, fi.Size())
			if err != nil {
				return fmt.Errorf("open parquet: %w", err)
			}
			schema := pf.Schema()
			cols := schema.Columns()

			type entry struct {
				Name     string `json:"name"`
				Type     string `json:"type"`
				Optional bool   `json:"optional"`
			}
			entries := make([]entry, len(cols))
			maxName := 0
			for i, path := range cols {
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
				entries[i] = entry{Name: key, Type: nodeTypeName(node), Optional: optional}
				if len(key) > maxName {
					maxName = len(key)
				}
			}

			if asJSON {
				PrintJSON(entries)
				return nil
			}

			for _, e := range entries {
				nullable := format.Yellow("required")
				if e.Optional {
					nullable = format.Gray("optional")
				}
				fmt.Printf("  %s  %s  %s\n",
					format.PadRight(e.Name, maxName),
					format.PadRight(format.Gray(e.Type), 24),
					nullable,
				)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}
