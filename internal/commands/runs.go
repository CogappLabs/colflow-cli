package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func NewRuns() *cli.Command {
	flags := append(CommonFlags(),
		&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 10, Usage: "Number of runs to show"},
		&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status (SUCCESS, FAILURE, STARTED, etc)"},
	)
	return &cli.Command{
		Name:  "runs",
		Usage: "List recent pipeline runs",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			limit := int(c.Int("limit"))
			if limit < 1 || limit > 100 {
				return fmt.Errorf("--limit must be 1..100")
			}
			runs, err := client.GetRuns(limit, c.String("status"))
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(runs)
				return nil
			}
			if len(runs) == 0 {
				fmt.Println(format.Gray("No runs found"))
				return nil
			}
			header := fmt.Sprintf("%s %s %s %s %s",
				format.PadRight("RUN ID", 38),
				format.PadRight("JOB", 30),
				format.PadRight("STATUS", 12),
				format.PadRight("STARTED", 12),
				format.PadRight("DURATION", 10),
			)
			fmt.Println(format.Bold(header))
			fmt.Println(format.Gray(strings.Repeat("─", len(header))))

			for _, r := range runs {
				dur := "—"
				if r.StartTime != 0 && r.EndTime != 0 {
					dur = fmt.Sprintf("%ds", int(r.EndTime-r.StartTime))
				} else if r.StartTime != 0 {
					dur = "running"
				}
				short := r.RunID
				if len(short) > 36 {
					short = short[:36]
				}
				fmt.Printf("%s %s %s %s %s\n",
					format.PadRight(short, 38),
					format.PadRight(r.JobName, 30),
					format.PadRight(format.ColorStatus(r.Status), 12),
					format.PadRight(format.TimeAgo(r.StartTime), 12),
					dur,
				)
			}
			return nil
		},
	}
}
