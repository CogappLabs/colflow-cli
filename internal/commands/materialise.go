package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func NewMaterialise() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "asset", Usage: "Single asset name to materialise"},
		&cli.StringFlag{Name: "assets", Usage: "Comma-separated asset names"},
	)
	return &cli.Command{
		Name:  "materialise",
		Usage: "Materialise specific assets by name",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			asset, assets := c.String("asset"), c.String("assets")
			var names []string
			if assets != "" {
				for _, s := range strings.Split(assets, ",") {
					names = append(names, strings.TrimSpace(s))
				}
			} else if asset != "" {
				names = []string{asset}
			} else {
				return fmt.Errorf(`Provide --asset <name> or --assets "name1,name2"`)
			}
			runID, err := client.LaunchAssetRun(names)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(map[string]any{"runId": runID, "assets": names})
				return nil
			}
			fmt.Println(format.Green(fmt.Sprintf("Materialising %s — run %s", strings.Join(names, ", "), format.Bold(runID))))
			return nil
		},
	}
}
