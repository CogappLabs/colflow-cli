package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewJobs() *cobra.Command {
	flags := &CommonFlags{}
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List all jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			jobs, err := client.GetJobs()
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(jobs)
				return nil
			}
			if len(jobs) == 0 {
				fmt.Println(format.Gray("No jobs found"))
				return nil
			}
			for _, j := range jobs {
				name := format.Bold(format.PadRight(j.Name, 30))
				if strings.HasPrefix(j.Name, "__") {
					name = format.Gray(format.PadRight(j.Name, 30))
				}
				desc := ""
				if j.Description != nil {
					desc = format.Gray(*j.Description)
				}
				fmt.Printf("%s %s\n", name, desc)
			}
			return nil
		},
	}
	AddCommon(cmd, flags)
	return cmd
}
