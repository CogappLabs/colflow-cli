package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

func NewAsset() *cli.Command {
	flags := append(CommonFlags(),
		&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: "Asset key path (e.g. raw_catalog)"},
	)
	return &cli.Command{
		Name:  "asset",
		Usage: "Show detailed info for a specific asset",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			k := c.String("key")
			if k == "" {
				v, ok := prompts.SelectAsset()
				if !ok {
					return nil
				}
				k = v
			}
			asset, err := client.GetAssetDetail(strings.Split(k, "/"))
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(asset)
				return nil
			}
			path := strings.Join(asset.AssetKey.Path, "/")
			fmt.Println(format.Bold(path))
			if asset.Description != nil {
				fmt.Println(format.Gray("  " + *asset.Description))
			}
			fmt.Println()

			fmt.Printf("  Group:        %s\n", strOr(asset.GroupName, "—"))
			fmt.Printf("  Compute:      %s\n", strOr(asset.ComputeKind, "—"))
			fmt.Printf("  Partitioned:  %s\n", yesNo(asset.IsPartitioned))
			staleColored := format.ColorStatus("FAILURE")
			if asset.StaleStatus == "FRESH" {
				staleColored = format.ColorStatus("SUCCESS")
			}
			fmt.Printf("  Stale:        %s (%s)\n", staleColored, asset.StaleStatus)
			if len(asset.Kinds) > 0 {
				fmt.Printf("  Kinds:        %s\n", strings.Join(asset.Kinds, ", "))
			}
			if len(asset.Tags) > 0 {
				tags := make([]string, len(asset.Tags))
				for i, t := range asset.Tags {
					tags[i] = t.Key + "=" + t.Value
				}
				fmt.Printf("  Tags:         %s\n", strings.Join(tags, ", "))
			}
			jobs := strings.Join(asset.JobNames, ", ")
			if jobs == "" {
				jobs = "—"
			}
			fmt.Printf("  Jobs:         %s\n", jobs)

			if len(asset.DependencyKeys) > 0 {
				fmt.Println()
				fmt.Println(format.Bold("  Dependencies (upstream):"))
				for _, dep := range asset.DependencyKeys {
					fmt.Printf("    ← %s\n", strings.Join(dep.Path, "/"))
				}
			}
			if len(asset.DependedByKeys) > 0 {
				fmt.Println()
				fmt.Println(format.Bold("  Dependents (downstream):"))
				for _, dep := range asset.DependedByKeys {
					fmt.Printf("    → %s\n", strings.Join(dep.Path, "/"))
				}
			}
			if len(asset.StaleCauses) > 0 {
				fmt.Println()
				fmt.Println(format.Bold("  Stale causes:"))
				for _, cs := range asset.StaleCauses {
					fmt.Printf("    %s: %s (%s)\n", format.Yellow(cs.Category), cs.Reason, strings.Join(cs.Key.Path, "/"))
				}
			}
			if len(asset.Materializations) > 0 {
				fmt.Println()
				fmt.Println(format.Bold("  Recent materializations:"))
				for _, m := range asset.Materializations {
					rid := m.RunID
					if len(rid) > 36 {
						rid = rid[:36]
					}
					fmt.Printf("    %s  %s\n", format.Gray(rid), format.TimeAgo(m.Timestamp))
					for _, me := range m.Metadata {
						fmt.Printf("      %s: %s\n", format.Cyan(me.Label), metaValue(me))
					}
				}
			}
			if asset.FreshnessInfo != nil && asset.FreshnessInfo.CurrentMinutesLate != nil {
				fmt.Println()
				mins := *asset.FreshnessInfo.CurrentMinutesLate
				if mins > 0 {
					fmt.Printf("  Freshness:    %s\n", format.Red(fmt.Sprintf("%.0fm late", mins)))
				} else {
					fmt.Printf("  Freshness:    %s\n", format.Green("on time"))
				}
			}
			return nil
		},
	}
}

func strOr(p *string, def string) string {
	if p == nil || *p == "" {
		return def
	}
	return *p
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func metaValue(m client.MetadataEntry) string {
	if m.Text != nil {
		return *m.Text
	}
	if m.Path != nil {
		return *m.Path
	}
	if m.IntValue != nil {
		return fmt.Sprintf("%d", *m.IntValue)
	}
	if m.FloatValue != nil {
		return fmt.Sprintf("%v", *m.FloatValue)
	}
	if m.BoolValue != nil {
		return fmt.Sprintf("%v", *m.BoolValue)
	}
	if m.JSONString != nil {
		return *m.JSONString
	}
	return m.TypeName
}
