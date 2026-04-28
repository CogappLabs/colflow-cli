package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewRuns() *cobra.Command {
	flags := &CommonFlags{}
	var limit int
	var status string
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "List recent pipeline runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			if limit < 1 || limit > 100 {
				return fmt.Errorf("--limit must be 1..100")
			}
			runs, err := client.GetRuns(limit, status)
			if err != nil {
				return err
			}
			if flags.JSON {
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
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of runs to show")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (SUCCESS, FAILURE, STARTED, etc)")
	AddCommon(cmd, flags)
	return cmd
}
