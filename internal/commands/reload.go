package commands

import (
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewReload() *cobra.Command {
	flags := &CommonFlags{}
	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload Dagster code location",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			r, err := client.ReloadLocation()
			if err != nil {
				return err
			}
			if flags.JSON {
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
	AddCommon(cmd, flags)
	return cmd
}
