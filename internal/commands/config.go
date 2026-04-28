package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func NewConfig() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "job", Aliases: []string{"j"}, Value: "full_pipeline", Usage: "Job name"},
	)
	return &cli.Command{
		Name:  "config",
		Usage: "Show the run config schema for a job",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			cfg, err := client.GetJobConfig(c.String("job"))
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(map[string]any{"job": cfg.JobName, "fields": cfg.Fields})
				return nil
			}
			fmt.Println(format.Bold(fmt.Sprintf("Config schema for %s\n", cfg.JobName)))
			if len(cfg.Fields) == 0 {
				fmt.Println(format.Gray("No config fields"))
				return nil
			}
			maxName := 0
			for _, f := range cfg.Fields {
				if len(f.Name) > maxName {
					maxName = len(f.Name)
				}
			}
			for _, f := range cfg.Fields {
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
}
