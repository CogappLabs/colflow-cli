package commands

import (
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

type stepSummary struct {
	StepKey  string
	Level    string
	HasError bool
}

func extractSteps(events []client.RunEvent) map[string]*stepSummary {
	steps := map[string]*stepSummary{}
	for _, e := range events {
		if e.StepKey == nil {
			continue
		}
		k := *e.StepKey
		s, ok := steps[k]
		if !ok {
			steps[k] = &stepSummary{StepKey: k, Level: e.Level, HasError: e.Level == "ERROR"}
		} else if e.Level == "ERROR" {
			s.HasError = true
		}
	}
	return steps
}

func formatDuration(start, end float64) string {
	if start == 0 || end == 0 {
		return "unknown"
	}
	seconds := int(end - start)
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
}

func NewDiff() *cobra.Command {
	flags := &CommonFlags{}
	var run1, run2 string
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare two runs side by side",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			if run1 == "" || run2 == "" {
				return fmt.Errorf("--run1 and --run2 required")
			}
			r1, err := client.GetRun(run1)
			if err != nil {
				return err
			}
			r2, err := client.GetRun(run2)
			if err != nil {
				return err
			}
			s1 := extractSteps(r1.Events)
			s2 := extractSteps(r2.Events)

			allKeys := map[string]bool{}
			for k := range s1 {
				allKeys[k] = true
			}
			for k := range s2 {
				allKeys[k] = true
			}

			type diffRow struct {
				Step string `json:"step"`
				Run1 string `json:"run1"`
				Run2 string `json:"run2"`
			}
			diffs := []diffRow{}
			statusOf := func(s *stepSummary) string {
				if s == nil {
					return "MISSING"
				}
				if s.HasError {
					return "FAILED"
				}
				return "OK"
			}
			for k := range allKeys {
				st1, st2 := statusOf(s1[k]), statusOf(s2[k])
				if st1 != st2 {
					diffs = append(diffs, diffRow{Step: k, Run1: st1, Run2: st2})
				}
			}

			if flags.JSON {
				PrintJSON(map[string]any{
					"run1": map[string]any{
						"runId":    r1.Run.RunID,
						"status":   r1.Run.Status,
						"duration": formatDuration(r1.Run.StartTime, r1.Run.EndTime),
						"steps":    r1.Run.Stats.StepsSucceeded + r1.Run.Stats.StepsFailed,
					},
					"run2": map[string]any{
						"runId":    r2.Run.RunID,
						"status":   r2.Run.Status,
						"duration": formatDuration(r2.Run.StartTime, r2.Run.EndTime),
						"steps":    r2.Run.Stats.StepsSucceeded + r2.Run.Stats.StepsFailed,
					},
					"differences": diffs,
				})
				return nil
			}

			id1 := r1.Run.RunID
			if len(id1) > 8 {
				id1 = id1[:8]
			}
			id2 := r2.Run.RunID
			if len(id2) > 8 {
				id2 = id2[:8]
			}

			fmt.Println(format.Bold("Run comparison\n"))
			fmt.Printf("  %s  %s  %s\n", format.PadRight("", 12), format.PadRight(id1, 20), id2)
			fmt.Printf("  %s  %s  %s\n", format.PadRight("Status", 12), format.PadRight(format.ColorStatus(r1.Run.Status), 20), format.ColorStatus(r2.Run.Status))
			fmt.Printf("  %s  %s  %s\n", format.PadRight("Job", 12), format.PadRight(r1.Run.JobName, 20), r2.Run.JobName)
			fmt.Printf("  %s  %s  %s\n", format.PadRight("Duration", 12), format.PadRight(formatDuration(r1.Run.StartTime, r1.Run.EndTime), 20), formatDuration(r2.Run.StartTime, r2.Run.EndTime))
			fmt.Printf("  %s  %s  %s\n", format.PadRight("Started", 12), format.PadRight(format.TimeAgo(r1.Run.StartTime), 20), format.TimeAgo(r2.Run.StartTime))
			fmt.Printf("  %s  %s  %d\n", format.PadRight("Succeeded", 12), format.PadRight(fmt.Sprintf("%d", r1.Run.Stats.StepsSucceeded), 20), r2.Run.Stats.StepsSucceeded)
			fmt.Printf("  %s  %s  %d\n", format.PadRight("Failed", 12), format.PadRight(fmt.Sprintf("%d", r1.Run.Stats.StepsFailed), 20), r2.Run.Stats.StepsFailed)
			fmt.Println()

			if len(diffs) == 0 {
				fmt.Println(format.Green("All steps match between runs"))
				return nil
			}
			fmt.Println(format.Bold(fmt.Sprintf("%d step(s) differ:\n", len(diffs))))
			maxStep := 0
			for _, d := range diffs {
				if len(d.Step) > maxStep {
					maxStep = len(d.Step)
				}
			}
			colourStep := func(s string) string {
				switch s {
				case "FAILED":
					return format.Red(s)
				case "MISSING":
					return format.Yellow(s)
				default:
					return format.Green(s)
				}
			}
			for _, d := range diffs {
				fmt.Printf("  %s  %s  %s\n",
					format.PadRight(d.Step, maxStep),
					format.PadRight(colourStep(d.Run1), 12),
					colourStep(d.Run2),
				)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&run1, "run1", "", "First run ID")
	cmd.Flags().StringVar(&run2, "run2", "", "Second run ID")
	AddCommon(cmd, flags)
	return cmd
}
