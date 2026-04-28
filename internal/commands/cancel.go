package commands

import (
	"context"
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

func NewCancel() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "id", Usage: "Run ID to cancel"},
	)
	return &cli.Command{
		Name:  "cancel",
		Usage: "Cancel a running or queued run",
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
			status, err := client.TerminateRun(runID)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(map[string]string{"runId": runID, "status": status})
				return nil
			}
			fmt.Println(format.Yellow(fmt.Sprintf("Cancelled run %s — status: %s", runID, status)))
			return nil
		},
	}
}
