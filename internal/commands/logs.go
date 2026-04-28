package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/spf13/cobra"
)

func NewLogs() *cobra.Command {
	flags := &CommonFlags{}
	var id, step, level, grep string
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View filtered logs for a run",
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
			events, err := client.GetRunLogs(runID, client.LogFilter{Step: step, Level: strings.ToUpper(level)})
			if err != nil {
				return err
			}
			if grep != "" {
				p := strings.ToLower(grep)
				filtered := events[:0]
				for _, e := range events {
					if strings.Contains(strings.ToLower(e.Message), p) {
						filtered = append(filtered, e)
					}
				}
				events = filtered
			}

			if flags.JSON {
				PrintJSON(events)
				return nil
			}
			if len(events) == 0 {
				fmt.Println(format.Gray("No matching log events"))
				return nil
			}
			for _, e := range events {
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
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Run ID")
	cmd.Flags().StringVarP(&step, "step", "s", "", "Filter by step key")
	cmd.Flags().StringVarP(&level, "level", "l", "", "Filter by level")
	cmd.Flags().StringVarP(&grep, "grep", "g", "", "Filter messages by substring")
	AddCommon(cmd, flags)
	return cmd
}
