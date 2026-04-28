package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/spf13/cobra"
)

const assetTemplate = `"""{{TITLE}}."""

import dagster as dg
{{IMPORTS}}


@dg.asset(group_name="{{GROUP}}", kinds={"polars"})
def {{NAME}}({{ARGS}}) -> pl.LazyFrame:
    """{{TITLE}}."""
    raise NotImplementedError("Implement {{NAME}}")


class {{CLASS}}Schema(pa.DataFrameModel):
    """Schema for {{NAME}}."""


@dg.asset_check(asset="{{NAME}}", name="schema", blocking=True)
def {{NAME}}_schema_check(df: pl.LazyFrame) -> dg.AssetCheckResult:
    """Check {{NAME}} matches its schema."""
    return validate_dataframe(df, {{CLASS}}Schema).to_asset_check_result(
        asset_key="{{NAME}}",
        check_name="schema",
    )
`

const testTemplate = `import polars as pl

from {{PKG}}.defs.assets.{{NAME}} import {{NAME}}


def test_{{NAME}}_produces_lazyframe() -> None:
    """{{NAME}} returns a Polars LazyFrame."""
    # TODO: build inputs that match upstream schema
    raise NotImplementedError("Implement test for {{NAME}}")
`

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

func NewNewAsset() *cobra.Command {
	var upstream, group, title string
	var withTest bool
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "new-asset <name>",
		Short: "Scaffold a new Dagster asset (Polars + Pandera schema + check)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !nameRegex.MatchString(name) {
				return fmt.Errorf("name must be snake_case (lowercase, digits, underscores), got: %s", name)
			}
			info, err := project.Detect(project.Cwd())
			if err != nil {
				return err
			}

			if title == "" {
				title = strings.ReplaceAll(name, "_", " ")
				title = strings.ToUpper(title[:1]) + title[1:]
			}

			args2 := []string{}
			imports := []string{
				"import pandera.polars as pa",
				"import polars as pl",
				"from collection_flow.support.polars.validation import validate_dataframe",
			}
			if upstream != "" {
				for _, dep := range strings.Split(upstream, ",") {
					dep = strings.TrimSpace(dep)
					if dep == "" {
						continue
					}
					args2 = append(args2, fmt.Sprintf("%s: pl.LazyFrame", dep))
				}
			}

			body := assetTemplate
			body = strings.ReplaceAll(body, "{{TITLE}}", title)
			body = strings.ReplaceAll(body, "{{IMPORTS}}", strings.Join(imports, "\n"))
			body = strings.ReplaceAll(body, "{{GROUP}}", group)
			body = strings.ReplaceAll(body, "{{NAME}}", name)
			body = strings.ReplaceAll(body, "{{ARGS}}", strings.Join(args2, ", "))
			body = strings.ReplaceAll(body, "{{CLASS}}", toClassName(name))

			assetPath := filepath.Join(info.AssetsDir, name+".py")

			fmt.Println(format.Bold("Project:"), info.Name)
			fmt.Println(format.Bold("Asset path:"), assetPath)

			if dryRun {
				fmt.Println()
				fmt.Println(format.Gray("--- " + assetPath + " ---"))
				fmt.Println(body)
				if withTest {
					testBody := strings.ReplaceAll(testTemplate, "{{PKG}}", info.PackageName)
					testBody = strings.ReplaceAll(testBody, "{{NAME}}", name)
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
				testBody := strings.ReplaceAll(testTemplate, "{{PKG}}", info.PackageName)
				testBody = strings.ReplaceAll(testBody, "{{NAME}}", name)
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
	cmd.Flags().StringVar(&upstream, "upstream", "", "Comma-separated upstream asset names (become function args)")
	cmd.Flags().StringVarP(&group, "group", "g", "transform", "Asset group name")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Asset title (default: derived from name)")
	cmd.Flags().BoolVar(&withTest, "test", true, "Also scaffold a tests/test_<name>.py file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print without writing")
	return cmd
}
