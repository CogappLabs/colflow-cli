package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewMaterialise() *cobra.Command {
	flags := &CommonFlags{}
	var asset, assets string
	cmd := &cobra.Command{
		Use:   "materialise",
		Short: "Materialise specific assets by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			var names []string
			if assets != "" {
				for _, s := range strings.Split(assets, ",") {
					names = append(names, strings.TrimSpace(s))
				}
			} else if asset != "" {
				names = []string{asset}
			} else {
				return fmt.Errorf(`Provide --asset <name> or --assets "name1,name2"`)
			}
			runID, err := client.LaunchAssetRun(names)
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(map[string]any{"runId": runID, "assets": names})
				return nil
			}
			fmt.Println(format.Green(fmt.Sprintf("Materialising %s — run %s", strings.Join(names, ", "), format.Bold(runID))))
			return nil
		},
	}
	cmd.Flags().StringVar(&asset, "asset", "", "Single asset name to materialise")
	cmd.Flags().StringVar(&assets, "assets", "", "Comma-separated asset names")
	AddCommon(cmd, flags)
	return cmd
}
