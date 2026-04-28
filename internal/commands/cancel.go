package commands

import (
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/spf13/cobra"
)

func NewCancel() *cobra.Command {
	flags := &CommonFlags{}
	var id string
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a running or queued run",
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
			status, err := client.TerminateRun(runID)
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(map[string]string{"runId": runID, "status": status})
				return nil
			}
			fmt.Println(format.Yellow(fmt.Sprintf("Cancelled run %s — status: %s", runID, status)))
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Run ID to cancel")
	AddCommon(cmd, flags)
	return cmd
}
