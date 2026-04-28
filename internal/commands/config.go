package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewConfig() *cobra.Command {
	flags := &CommonFlags{}
	var job string
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show the run config schema for a job",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			c, err := client.GetJobConfig(job)
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(map[string]any{"job": c.JobName, "fields": c.Fields})
				return nil
			}
			fmt.Println(format.Bold(fmt.Sprintf("Config schema for %s\n", c.JobName)))
			if len(c.Fields) == 0 {
				fmt.Println(format.Gray("No config fields"))
				return nil
			}
			maxName := 0
			for _, f := range c.Fields {
				if len(f.Name) > maxName {
					maxName = len(f.Name)
				}
			}
			for _, f := range c.Fields {
				required := format.Gray("optional")
				if f.IsRequired {
					required = format.Red("required")
				}
				def := ""
				if f.DefaultValueAsJSON != nil {
					def = format.Cyan(" = " + *f.DefaultValueAsJSON)
				}
				fmt.Printf("  %s  %s  %s%s\n",
					format.PadRight(f.Name, maxName),
					format.Gray(f.ConfigTypeKey),
					required,
					def,
				)
				if f.Description != nil && *f.Description != "" {
					fmt.Println(format.Gray("  " + strings.Repeat(" ", maxName) + "  " + *f.Description))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&job, "job", "j", "full_pipeline", "Job name")
	AddCommon(cmd, flags)
	return cmd
}
