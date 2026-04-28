package commands

import (
	"context"
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func NewReload() *cli.Command {
	return &cli.Command{
		Name:  "reload",
		Usage: "Reload Dagster code location",
		Flags: CommonFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			r, err := client.ReloadLocation()
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(r)
				return nil
			}
			switch r.Status {
			case "LOADED":
				fmt.Println(format.Green("Code location reloaded successfully"))
			case "ERROR":
				fmt.Println(format.Red("Reload failed: " + r.Message))
			default:
				fmt.Println(format.Yellow("Reload status: " + r.Status))
				if r.Message != "" {
					fmt.Println(format.Gray("  " + r.Message))
				}
			}
			return nil
		},
	}
}
