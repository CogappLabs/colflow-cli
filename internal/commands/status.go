package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func NewStatus() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Quick pipeline health summary",
		Flags: CommonFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			rc, err := client.GetRunCounts()
			if err != nil {
				return err
			}
			assets, err := client.GetAssets()
			if err != nil {
				return err
			}
			materialized := 0
			for _, a := range assets {
				if len(a.Materializations) > 0 {
					materialized++
				}
			}

			if c.Bool("json") {
				PrintJSON(map[string]any{
					"latest": rc.Latest,
					"counts": rc.Counts,
					"assets": map[string]int{"total": len(assets), "materialized": materialized},
				})
				return nil
			}

			if rc.Latest == nil {
				fmt.Println(format.Yellow("No runs found"))
				return nil
			}
			ago := format.TimeAgo(rc.Latest.StartTime)
			switch rc.Latest.Status {
			case "SUCCESS":
				fmt.Println(format.Green(fmt.Sprintf("Pipeline OK — last run %s (SUCCESS), %d/%d assets materialized", ago, materialized, len(assets))))
			case "FAILURE":
				fmt.Println(format.Red(fmt.Sprintf("FAILURE — %s failed %s", rc.Latest.JobName, ago)))
			default:
				fmt.Println(format.Yellow(fmt.Sprintf("%s — %s %s", rc.Latest.Status, rc.Latest.JobName, ago)))
			}

			parts := make([]string, 0, len(rc.Counts))
			for s, n := range rc.Counts {
				parts = append(parts, fmt.Sprintf("%s: %d", s, n))
			}
			fmt.Println(format.Gray("  Recent: " + strings.Join(parts, ", ")))
			return nil
		},
	}
}
