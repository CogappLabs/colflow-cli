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
	"github.com/spf13/cobra"
)

func NewTail() *cobra.Command {
	flags := &CommonFlags{}
	var id string
	var interval int
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Live-follow a running job's logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			runID := id
			if runID == "" {
				v, ok := prompts.SelectRun()
				if !ok {
					return nil
				}
				runID = v
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			fmt.Println(format.Gray(fmt.Sprintf("Tailing run %s (Ctrl+C to stop)...\n", runID)))

			seen := 0
			terminal := map[string]bool{"SUCCESS": true, "FAILURE": true, "CANCELED": true}

			for {
				if ctx.Err() != nil {
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
				case <-ctx.Done():
					return nil
				case <-time.After(time.Duration(interval) * time.Second):
				}
			}
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Run ID to follow")
	cmd.Flags().IntVarP(&interval, "interval", "i", 3, "Poll interval in seconds")
	cmd.Flags().StringVarP(&flags.URL, "url", "u", "", "Dagster base URL")
	cmd.Flags().StringVarP(&flags.Auth, "auth", "a", "", "HTTP basic auth")
	return cmd
}
