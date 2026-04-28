package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

func NewErrors() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "id", Usage: "Run ID"},
	)
	return &cli.Command{
		Name:  "errors",
		Usage: "Get Python tracebacks from a failed run",
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
			failures, err := client.GetRunErrors(runID)
			if err != nil {
				return err
			}
			if c.Bool("json") {
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
					for _, ca := range f.Error.Causes {
						fmt.Println(format.Red("    " + ca.Message))
						cs := ca.Stack
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
}
