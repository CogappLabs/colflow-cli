package commands

import (
	"context"
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

func NewRun() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "id", Usage: "Run ID"},
		&cli.IntFlag{Name: "events", Aliases: []string{"e"}, Value: 20, Usage: "Number of log events to show"},
	)
	return &cli.Command{
		Name:  "run",
		Usage: "Get details and logs for a specific run",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			runID := c.String("id")
			if runID == "" {
				v, ok := prompts.SelectRun()
				if !ok {
					return nil
				}
				runID = v
			}
			d, err := client.GetRun(runID)
			if err != nil {
				return err
			}
			events := int(c.Int("events"))
			tail := d.Events
			if len(tail) > events {
				tail = tail[len(tail)-events:]
			}

			if c.Bool("json") {
				PrintJSON(map[string]any{"run": d.Run, "events": tail})
				return nil
			}

			fmt.Println(format.Bold("Run " + d.Run.RunID))
			fmt.Printf("  Job:     %s\n", d.Run.JobName)
			fmt.Printf("  Status:  %s\n", format.ColorStatus(d.Run.Status))
			fmt.Printf("  Started: %s\n", format.TimeAgo(d.Run.StartTime))
			failed := fmt.Sprintf("%d failed", d.Run.Stats.StepsFailed)
			if d.Run.Stats.StepsFailed > 0 {
				failed = format.Red(failed)
			}
			fmt.Printf("  Steps:   %s  %s\n", format.Green(fmt.Sprintf("%d succeeded", d.Run.Stats.StepsSucceeded)), failed)
			fmt.Println()

			if len(tail) > 0 {
				fmt.Println(format.Bold("Recent events:"))
				for _, e := range tail {
					step := ""
					if e.StepKey != nil {
						step = format.Cyan("[" + *e.StepKey + "]")
					}
					level := format.Gray(e.Level)
					if e.Level == "ERROR" {
						level = format.Red(e.Level)
					}
					fmt.Printf("  %s %s %s\n", level, step, e.Message)
				}
			}
			return nil
		},
	}
}
