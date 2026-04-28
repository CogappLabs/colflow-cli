package commands

import (
	"fmt"
	"os"

	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

func NewSchema() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "schema [file.parquet | asset_name]",
		Short: "List the schema of a parquet file (tree view)",
		Long:  "Show columns + types as a tree. Parquet list/map wrappers (list.element, key_value) are collapsed. Bare names resolve to <project>/output/<name>.parquet.",
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
			entries := FlattenSchema(pf.Schema())

			if asJSON {
				PrintJSON(entries)
				return nil
			}
			PrintSchemaTree(entries)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}
