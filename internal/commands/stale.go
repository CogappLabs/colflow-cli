package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewStale() *cobra.Command {
	flags := &CommonFlags{}
	cmd := &cobra.Command{
		Use:   "stale",
		Short: "List stale assets that need re-materialisation",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			stale, err := client.GetStaleAssets()
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(stale)
				return nil
			}
			if len(stale) == 0 {
				fmt.Println(format.Green("All assets are fresh"))
				return nil
			}
			fmt.Println(format.Bold(fmt.Sprintf("%d stale asset(s):\n", len(stale))))

			maxName := 0
			for _, a := range stale {
				n := len(strings.Join(a.AssetKey.Path, "/"))
				if n > maxName {
					maxName = n
				}
			}
			for _, a := range stale {
				name := strings.Join(a.AssetKey.Path, "/")
				group := "ungrouped"
				if a.GroupName != nil {
					group = *a.GroupName
				}
				lastMat := "never"
				if len(a.Materializations) > 0 {
					lastMat = format.TimeAgo(a.Materializations[0].Timestamp)
				}
				status := format.Red(a.StaleStatus)
				if a.StaleStatus == "STALE" {
					status = format.Yellow(a.StaleStatus)
				}
				fmt.Printf("  %s  %s  %s  %s\n",
					status,
					format.PadRight(name, maxName),
					format.Gray(group),
					format.Gray("last: "+lastMat),
				)
				for _, c := range a.StaleCauses {
					fmt.Println(format.Gray(fmt.Sprintf("         %s: %s (%s)", c.Category, c.Reason, strings.Join(c.Key.Path, "/"))))
				}
			}
			return nil
		},
	}
	AddCommon(cmd, flags)
	return cmd
}
