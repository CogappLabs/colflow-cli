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

func NewLogs() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "id", Usage: "Run ID"},
		&cli.StringFlag{Name: "step", Aliases: []string{"s"}, Usage: "Filter by step key"},
		&cli.StringFlag{Name: "level", Aliases: []string{"l"}, Usage: "Filter by level"},
		&cli.StringFlag{Name: "grep", Aliases: []string{"g"}, Usage: "Filter messages by substring"},
	)
	return &cli.Command{
		Name:  "logs",
		Usage: "View filtered logs for a run",
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
			events, err := client.GetRunLogs(runID, client.LogFilter{Step: c.String("step"), Level: strings.ToUpper(c.String("level"))})
			if err != nil {
				return err
			}
			grep := c.String("grep")
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

			if c.Bool("json") {
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
}
