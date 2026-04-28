package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func NewJobs() *cli.Command {
	return &cli.Command{
		Name:  "jobs",
		Usage: "List all jobs",
		Flags: CommonFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			jobs, err := client.GetJobs()
			if err != nil {
				return err
			}
			if c.Bool("json") {
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
}
