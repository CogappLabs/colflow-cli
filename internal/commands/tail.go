package commands

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

func NewTail() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "id", Usage: "Run ID to follow"},
		&cli.IntFlag{Name: "interval", Aliases: []string{"i"}, Value: 3, Usage: "Poll interval in seconds"},
	)
	return &cli.Command{
		Name:  "tail",
		Usage: "Live-follow a running job's logs",
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

			sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			fmt.Println(format.Gray(fmt.Sprintf("Tailing run %s (Ctrl+C to stop)...\n", runID)))

			seen := 0
			terminal := map[string]bool{"SUCCESS": true, "FAILURE": true, "CANCELED": true}
			interval := int(c.Int("interval"))

			for {
				if sigCtx.Err() != nil {
					return nil
				}
				d, err := client.GetRun(runID)
				if err != nil {
					return err
				}
				filtered := []client.RunEvent{}
				for _, e := range d.Events {
					if e.Message != "" {
						filtered = append(filtered, e)
					}
				}
				newEvents := filtered[seen:]
				for _, e := range newEvents {
					stepStr := ""
					if e.StepKey != nil {
						stepStr = format.Cyan("[" + *e.StepKey + "]")
					}
					lvl := format.Gray(e.Level)
					if e.Level == "ERROR" {
						lvl = format.Red(e.Level)
					} else if e.Level == "WARNING" {
						lvl = format.Yellow(e.Level)
					}
					fmt.Printf("%s %s %s\n", lvl, stepStr, e.Message)
				}
				seen += len(newEvents)

				if terminal[d.Run.Status] {
					fmt.Println()
					fmt.Printf("Run %s — %d steps succeeded, %d failed (%s)\n",
						format.ColorStatus(d.Run.Status),
						d.Run.Stats.StepsSucceeded,
						d.Run.Stats.StepsFailed,
						format.TimeAgo(d.Run.StartTime),
					)
					return nil
				}
				select {
				case <-sigCtx.Done():
					return nil
				case <-time.After(time.Duration(interval) * time.Second):
				}
			}
		},
	}
}
