package commands

import (
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/spf13/cobra"
)

func NewRun() *cobra.Command {
	flags := &CommonFlags{}
	var id string
	var events int
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Get details and logs for a specific run",
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
			d, err := client.GetRun(runID)
			if err != nil {
				return err
			}
			tail := d.Events
			if len(tail) > events {
				tail = tail[len(tail)-events:]
			}

			if flags.JSON {
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
	cmd.Flags().StringVar(&id, "id", "", "Run ID")
	cmd.Flags().IntVarP(&events, "events", "e", 20, "Number of log events to show")
	AddCommon(cmd, flags)
	return cmd
}
