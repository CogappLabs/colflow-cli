package commands

import (
	"context"
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

func NewLaunch() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "job", Aliases: []string{"j"}, Usage: "Job name to launch"},
	)
	return &cli.Command{
		Name:  "launch",
		Usage: "Launch a job run",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			name := c.String("job")
			if name == "" {
				v, ok := prompts.SelectJob()
				if !ok {
					return nil
				}
				name = v
			}
			runID, err := client.LaunchRun(name)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(map[string]string{"runId": runID, "job": name})
				return nil
			}
			fmt.Println(format.Green(fmt.Sprintf("Launched %s — run %s", name, format.Bold(runID))))
			return nil
		},
	}
}
