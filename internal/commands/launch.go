package commands

import (
	"fmt"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/spf13/cobra"
)

func NewLaunch() *cobra.Command {
	flags := &CommonFlags{}
	var job string
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a job run",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			name := job
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
			if flags.JSON {
				PrintJSON(map[string]string{"runId": runID, "job": name})
				return nil
			}
			fmt.Println(format.Green(fmt.Sprintf("Launched %s — run %s", name, format.Bold(runID))))
			return nil
		},
	}
	cmd.Flags().StringVarP(&job, "job", "j", "", "Job name to launch")
	AddCommon(cmd, flags)
	return cmd
}
