package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

// NewRecheck launches a run that evaluates one or more asset checks
// without rematerialising the underlying assets. Useful for clearing
// red asset-check status in the UI after a schema or check-code fix
// where the asset's parquet output is unchanged.
//
// Selector format: "asset_name:check_name". Asset paths with multiple
// segments use slashes — "media_neon/image_metadata_row:row_count".
//
// Examples:
//
//	colflow recheck --check objects:objects_schema_check
//	colflow recheck --checks objects:objects_schema_check,objects:objects_unique_id
func NewRecheck() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "check", Usage: `Single check selector — "asset:check_name"`},
		&cli.StringFlag{Name: "checks", Usage: `Comma-separated selectors — "asset1:check1,asset2:check2"`},
	)
	return &cli.Command{
		Name:  "recheck",
		Usage: "Re-run asset checks (without rematerialising the asset)",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			single, multi := c.String("check"), c.String("checks")
			var raw []string
			if multi != "" {
				for _, s := range strings.Split(multi, ",") {
					raw = append(raw, strings.TrimSpace(s))
				}
			} else if single != "" {
				raw = []string{single}
			} else {
				return fmt.Errorf(`Provide --check <asset:check_name> or --checks "a1:c1,a2:c2"`)
			}

			selections := make([]client.AssetCheckSelection, 0, len(raw))
			for _, s := range raw {
				parts := strings.SplitN(s, ":", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid selector %q — expected asset:check_name", s)
				}
				assetPath := strings.Split(parts[0], "/")
				selections = append(selections, client.AssetCheckSelection{
					AssetPath: assetPath,
					CheckName: parts[1],
				})
			}

			runID, err := client.LaunchAssetCheckRun(selections)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(map[string]any{"runId": runID, "checks": raw})
				return nil
			}
			fmt.Println(format.Green(fmt.Sprintf("Re-checking %s — run %s", strings.Join(raw, ", "), format.Bold(runID))))
			return nil
		},
	}
}
