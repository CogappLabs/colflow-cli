package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewAssets() *cobra.Command {
	flags := &CommonFlags{}
	cmd := &cobra.Command{
		Use:   "assets",
		Short: "List all assets with last materialization",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			assets, err := client.GetAssets()
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(assets)
				return nil
			}
			if len(assets) == 0 {
				fmt.Println(format.Gray("No assets found"))
				return nil
			}
			header := fmt.Sprintf("%s %s %s %s",
				format.PadRight("ASSET", 40),
				format.PadRight("GROUP", 20),
				format.PadRight("LAST MATERIALIZED", 20),
				format.PadRight("STATUS", 10),
			)
			fmt.Println(format.Bold(header))
			fmt.Println(format.Gray(strings.Repeat("─", len(header))))

			for _, a := range assets {
				path := strings.Join(a.AssetKey.Path, "/")
				group := "—"
				if a.GroupName != nil {
					group = *a.GroupName
				}
				lastMat := "never"
				status := format.Gray("STALE")
				if len(a.Materializations) > 0 {
					lastMat = format.TimeAgo(a.Materializations[0].Timestamp)
					status = format.Green("OK")
				}
				fmt.Printf("%s %s %s %s\n",
					format.PadRight(path, 40),
					format.PadRight(group, 20),
					format.PadRight(lastMat, 20),
					status,
				)
			}
			return nil
		},
	}
	AddCommon(cmd, flags)
	return cmd
}
