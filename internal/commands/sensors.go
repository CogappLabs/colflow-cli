package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

func NewSensors() *cobra.Command {
	flags := &CommonFlags{}
	cmd := &cobra.Command{
		Use:   "sensors",
		Short: "Show sensor status and recent ticks",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Apply()
			sensors, err := client.GetSensors()
			if err != nil {
				return err
			}
			if flags.JSON {
				PrintJSON(sensors)
				return nil
			}
			if len(sensors) == 0 {
				fmt.Println(format.Gray("No sensors found"))
				return nil
			}
			for _, s := range sensors {
				colour := format.Yellow
				switch s.Status {
				case "RUNNING":
					colour = format.Green
				case "STOPPED":
					colour = format.Red
				}
				fmt.Printf("%s %s\n", format.Bold(format.PadRight(s.Name, 50)), colour(s.Status))
				if s.NextTick != nil {
					fmt.Println(format.Gray("  Next tick: " + format.TimeAgo(s.NextTick.Timestamp)))
				}
				for _, t := range s.Ticks {
					ts := format.Gray(t.Status)
					if t.Status == "SUCCESS" {
						ts = format.Green(t.Status)
					} else if t.Status == "FAILURE" {
						ts = format.Red(t.Status)
					}
					errMsg := ""
					if t.Error != nil {
						first := strings.SplitN(t.Error.Message, "\n", 2)[0]
						errMsg = format.Red(" — " + first)
					}
					fmt.Printf("  %s %s%s\n", ts, format.Gray(format.TimeAgo(t.Timestamp)), errMsg)
				}
				fmt.Println()
			}
			return nil
		},
	}
	AddCommon(cmd, flags)
	return cmd
}
