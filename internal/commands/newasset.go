package commands

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/urfave/cli/v3"
)

//go:embed templates/asset.py.tmpl
var assetTmplSrc string

//go:embed templates/test.py.tmpl
var testTmplSrc string

var assetTmpl = template.Must(template.New("asset").Parse(assetTmplSrc))
var testTmpl = template.Must(template.New("test").Parse(testTmplSrc))

type assetData struct {
	Name      string
	Class     string
	Title     string
	Group     string
	Kinds     []string
	Upstream  []string
	IsExtract bool
}

type testData struct {
	Name    string
	Package string
}

func renderTemplate(t *template.Template, data any) string {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

type knownAsset struct {
	Name  string
	Group string
}

// fetchKnownAssets queries Dagster live; on failure, falls back to filenames in defs/assets.
func fetchKnownAssets(assetsDir string) (assets []knownAsset, fromDagster bool) {
	graph, err := client.GetAssetGraph()
	if err == nil && len(graph) > 0 {
		out := make([]knownAsset, 0, len(graph))
		for _, n := range graph {
			name := strings.Join(n.AssetKey.Path, "/")
			group := ""
			if n.GroupName != nil {
				group = *n.GroupName
			}
			out = append(out, knownAsset{Name: name, Group: group})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Group != out[j].Group {
				return out[i].Group < out[j].Group
			}
			return out[i].Name < out[j].Name
		})
		return out, true
	}
	// Fallback
	names := listExistingAssets(assetsDir)
	out := make([]knownAsset, len(names))
	for i, n := range names {
		out[i] = knownAsset{Name: n}
	}
	return out, false
}

func mostCommonGroup(assets []knownAsset) string {
	counts := map[string]int{}
	best := ""
	bestN := 0
	for _, a := range assets {
		if a.Group == "" {
			continue
		}
		counts[a.Group]++
		if counts[a.Group] > bestN {
			bestN = counts[a.Group]
			best = a.Group
		}
	}
	if best == "" {
		return "transform"
	}
	return best
}

// pickUpstream returns selected asset names. Input "1,3,5" or comma-separated names.
func pickUpstream(assets []knownAsset) []string {
	if len(assets) == 0 {
		return strings.FieldsFunc(prompts.Ask("Upstream assets (comma-separated names, blank for none)", ""), func(r rune) bool { return r == ',' })
	}
	fmt.Println(format.Gray("Existing assets:"))
	for i, a := range assets {
		grp := ""
		if a.Group != "" {
			grp = format.Gray(" (" + a.Group + ")")
		}
		fmt.Printf("  %d) %s%s\n", i+1, a.Name, grp)
	}
	raw := prompts.Ask("Upstream (numbers or names, comma-separated, blank for none)", "")
	if raw == "" {
		return nil
	}
	picks := []string{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if idx, err := strconv.Atoi(p); err == nil && idx >= 1 && idx <= len(assets) {
			picks = append(picks, assets[idx-1].Name)
			continue
		}
		picks = append(picks, p)
	}
	return picks
}

func listExistingAssets(assetsDir string) []string {
	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".py") && name != "__init__.py" {
			out = append(out, strings.TrimSuffix(name, ".py"))
		}
	}
	return out
}

var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func toClassName(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func NewNewAsset() *cli.Command {
	return &cli.Command{
		Name:        "new-asset",
		Usage:       "Scaffold a new Dagster asset (Polars + Pandera schema + check)",
		Description: "Scaffold a new asset. With no name, runs interactively; otherwise uses flags.",
		ArgsUsage:   "[name]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "upstream", Usage: "Comma-separated upstream asset names (become function args)"},
			&cli.StringFlag{Name: "group", Aliases: []string{"g"}, Value: "transform", Usage: "Asset group name"},
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "Asset title (default: derived from name)"},
			&cli.BoolFlag{Name: "test", Value: true, Usage: "Also scaffold a tests/test_<name>.py file"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Print without writing"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() > 1 {
				return fmt.Errorf("new-asset takes at most 1 argument")
			}
			info, err := project.Detect(project.Cwd())
			if err != nil {
				return err
			}

			upstream := c.String("upstream")
			group := c.String("group")
			title := c.String("title")
			withTest := c.Bool("test")
			dryRun := c.Bool("dry-run")

			interactive := c.NArg() == 0
			var name string
			if !interactive {
				name = c.Args().Get(0)
			} else {
				fmt.Printf("%s %s\n\n", format.Bold("New asset for"), info.Name)
				name = prompts.Ask("Asset name (snake_case)", "")
				if name == "" {
					return fmt.Errorf("asset name required")
				}

				known, fromDagster := fetchKnownAssets(info.AssetsDir)
				if fromDagster {
					fmt.Println(format.Gray("(asset list from running Dagster)"))
				} else if len(known) > 0 {
					fmt.Println(format.Gray("(asset list from defs/assets/ — Dagster not reachable)"))
				}

				picks := pickUpstream(known)
				upstream = strings.Join(picks, ",")

				defaultGroup := group
				if defaultGroup == "transform" {
					defaultGroup = mostCommonGroup(known)
				}
				group = prompts.Ask("Group", defaultGroup)

				defaultTitle := strings.ReplaceAll(name, "_", " ")
				if len(defaultTitle) > 0 {
					defaultTitle = strings.ToUpper(defaultTitle[:1]) + defaultTitle[1:]
				}
				title = prompts.Ask("Title", defaultTitle)
				withTest = prompts.Confirm("Generate test stub?", withTest)
			}

			if !nameRegex.MatchString(name) {
				return fmt.Errorf("name must be snake_case (lowercase, digits, underscores), got: %s", name)
			}

			if title == "" {
				title = strings.ReplaceAll(name, "_", " ")
				title = strings.ToUpper(title[:1]) + title[1:]
			}

			isExtract := group == "extract"
			upstreams := []string{}
			if upstream != "" {
				for _, dep := range strings.Split(upstream, ",") {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						upstreams = append(upstreams, dep)
					}
				}
			}
			kinds := []string{"polars"}
			if isExtract {
				kinds = []string{"http"}
			}

			data := assetData{
				Name:      name,
				Class:     toClassName(name),
				Title:     title,
				Group:     group,
				Kinds:     kinds,
				Upstream:  upstreams,
				IsExtract: isExtract,
			}
			body := renderTemplate(assetTmpl, data)

			assetPath := filepath.Join(info.AssetsDir, name+".py")

			fmt.Println(format.Bold("Project:"), info.Name)
			fmt.Println(format.Bold("Asset path:"), assetPath)

			if dryRun {
				fmt.Println()
				fmt.Println(format.Gray("--- " + assetPath + " ---"))
				fmt.Println(body)
				if withTest {
					testBody := renderTemplate(testTmpl, testData{Name: name, Package: info.PackageName})
					testPath := filepath.Join(info.Root, "tests", "test_"+name+".py")
					fmt.Println(format.Gray("--- " + testPath + " ---"))
					fmt.Println(testBody)
				}
				return nil
			}

			if _, err := os.Stat(assetPath); err == nil {
				return fmt.Errorf("asset file already exists: %s", assetPath)
			}
			if err := os.MkdirAll(info.AssetsDir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(assetPath, []byte(body), 0o644); err != nil {
				return err
			}
			fmt.Println(format.Green("Created " + assetPath))

			if withTest {
				testBody := renderTemplate(testTmpl, testData{Name: name, Package: info.PackageName})
				testsDir := filepath.Join(info.Root, "tests")
				if err := os.MkdirAll(testsDir, 0o755); err != nil {
					return err
				}
				testPath := filepath.Join(testsDir, "test_"+name+".py")
				if _, err := os.Stat(testPath); err == nil {
					fmt.Println(format.Yellow("Skipped (exists): " + testPath))
				} else if err := os.WriteFile(testPath, []byte(testBody), 0o644); err != nil {
					return err
				} else {
					fmt.Println(format.Green("Created " + testPath))
				}
			}
			return nil
		},
	}
}
