package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/spf13/cobra"
)

func NewErrors() *cobra.Command {
	flags := &CommonFlags{}
	var id string
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Get Python tracebacks from a failed run",
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
			failures, err := client.GetRunErrors(runID)
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(failures)
				return nil
			}
			if len(failures) == 0 {
				fmt.Println(format.Gray("No step failures found"))
				return nil
			}
			for _, f := range failures {
				fmt.Println(format.BoldRed("Step: " + f.StepKey))
				fmt.Println(format.Red("  " + f.Error.Message))
				fmt.Println()

				stack := f.Error.Stack
				start := 0
				if len(stack) > 20 {
					fmt.Println(format.Gray(fmt.Sprintf("  ... (%d lines omitted)", len(stack)-20)))
					start = len(stack) - 20
				}
				for _, line := range stack[start:] {
					fmt.Println(format.Gray("  " + strings.TrimRight(line, " \t\n")))
				}
				if len(f.Error.Causes) > 0 {
					fmt.Println()
					fmt.Println(format.BoldRed("  Caused by:"))
					for _, c := range f.Error.Causes {
						fmt.Println(format.Red("    " + c.Message))
						cs := c.Stack
						cstart := 0
						if len(cs) > 10 {
							cstart = len(cs) - 10
						}
						for _, line := range cs[cstart:] {
							fmt.Println(format.Gray("    " + strings.TrimRight(line, " \t\n")))
						}
					}
				}
				fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Run ID")
	AddCommon(cmd, flags)
	return cmd
}
