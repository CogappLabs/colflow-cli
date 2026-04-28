package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/urfave/cli/v3"
)

func topoSort(nodes []client.AssetGraphNode) []client.AssetGraphNode {
	byKey := map[string]client.AssetGraphNode{}
	for _, n := range nodes {
		byKey[strings.Join(n.AssetKey.Path, "/")] = n
	}
	visited := map[string]bool{}
	result := []client.AssetGraphNode{}

	var visit func(string)
	visit = func(k string) {
		if visited[k] {
			return
		}
		visited[k] = true
		n, ok := byKey[k]
		if !ok {
			return
		}
		for _, dep := range n.DependencyKeys {
			visit(strings.Join(dep.Path, "/"))
		}
		result = append(result, n)
	}
	for _, n := range nodes {
		visit(strings.Join(n.AssetKey.Path, "/"))
	}
	return result
}

func NewGraph() *cli.Command {
	return &cli.Command{
		Name:  "graph",
		Usage: "Show the asset dependency graph",
		Flags: CommonFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			ApplyCommon(c)
			nodes, err := client.GetAssetGraph()
			if err != nil {
				return err
			}
			if c.Bool("json") {
				PrintJSON(nodes)
				return nil
			}
			sorted := topoSort(nodes)
			depths := map[string]int{}
			for _, n := range sorted {
				k := strings.Join(n.AssetKey.Path, "/")
				max := -1
				for _, d := range n.DependencyKeys {
					if v, ok := depths[strings.Join(d.Path, "/")]; ok && v > max {
						max = v
					}
				}
				if len(n.DependencyKeys) == 0 {
					depths[k] = 0
				} else {
					depths[k] = max + 1
				}
			}

			groups := map[string][]client.AssetGraphNode{}
			order := []string{}
			for _, n := range sorted {
				g := "ungrouped"
				if n.GroupName != nil {
					g = *n.GroupName
				}
				if _, ok := groups[g]; !ok {
					order = append(order, g)
				}
				groups[g] = append(groups[g], n)
			}

			for _, g := range order {
				fmt.Println(format.Bold("[" + g + "]"))
				for _, n := range groups[g] {
					k := strings.Join(n.AssetKey.Path, "/")
					depth := depths[k]
					indent := strings.Repeat("  ", depth+1)
					var deps string
					if len(n.DependencyKeys) > 0 {
						names := make([]string, len(n.DependencyKeys))
						for i, d := range n.DependencyKeys {
							names[i] = strings.Join(d.Path, "/")
						}
						deps = format.Gray(" ← " + strings.Join(names, ", "))
					} else {
						deps = format.Gray(" (root)")
					}
					var down string
					if len(n.DependedByKeys) > 0 {
						names := make([]string, len(n.DependedByKeys))
						for i, d := range n.DependedByKeys {
							names[i] = strings.Join(d.Path, "/")
						}
						down = format.Cyan(" → " + strings.Join(names, ", "))
					} else {
						down = format.Yellow(" (terminal)")
					}
					fmt.Printf("%s%s%s%s\n", indent, k, deps, down)
				}
				fmt.Println()
			}
			return nil
		},
	}
}
