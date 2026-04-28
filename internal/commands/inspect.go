package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/lukew-cogapp/colflow-cli/internal/prompts"
	"github.com/parquet-go/parquet-go"
	"github.com/spf13/cobra"
)

// printDagsterSection looks up parquet basename as a Dagster asset and prints
// metadata if found. Silent on failure.
func printDagsterSection(parquetPath string) {
	base := strings.TrimSuffix(filepath.Base(parquetPath), ".parquet")
	detail, err := client.GetAssetDetail([]string{base})
	if err != nil {
		return
	}
	fmt.Println()
	fmt.Println(format.Bold("Dagster:"))

	group := "—"
	if detail.GroupName != nil && *detail.GroupName != "" {
		group = *detail.GroupName
	}
	fmt.Printf("  %-12s %s\n", "Asset:", format.Cyan(strings.Join(detail.AssetKey.Path, "/")))
	fmt.Printf("  %-12s %s\n", "Group:", group)
	if detail.ComputeKind != nil && *detail.ComputeKind != "" {
		fmt.Printf("  %-12s %s\n", "Compute:", *detail.ComputeKind)
	}
	if len(detail.Kinds) > 0 {
		fmt.Printf("  %-12s %s\n", "Kinds:", strings.Join(detail.Kinds, ", "))
	}

	staleColoured := format.ColorStatus("FAILURE")
	if detail.StaleStatus == "FRESH" {
		staleColoured = format.ColorStatus("SUCCESS")
	}
	fmt.Printf("  %-12s %s (%s)\n", "Stale:", staleColoured, detail.StaleStatus)

	if len(detail.Materializations) > 0 {
		m := detail.Materializations[0]
		rid := m.RunID
		if len(rid) > 8 {
			rid = rid[:8]
		}
		fmt.Printf("  %-12s %s  %s  %s\n", "Last mat:",
			format.FormatTimestamp(m.Timestamp),
			format.Gray("("+format.TimeAgo(m.Timestamp)+")"),
			format.Gray("run "+rid),
		)
	}

	if len(detail.JobNames) > 0 {
		fmt.Printf("  %-12s %s\n", "Jobs:", strings.Join(detail.JobNames, ", "))
	}

	if len(detail.DependencyKeys) > 0 {
		deps := make([]string, len(detail.DependencyKeys))
		for i, d := range detail.DependencyKeys {
			deps[i] = strings.Join(d.Path, "/")
		}
		fmt.Printf("  %-12s %s\n", "Upstream:", format.Gray(strings.Join(deps, ", ")))
	}
	if len(detail.DependedByKeys) > 0 {
		dl := make([]string, len(detail.DependedByKeys))
		for i, d := range detail.DependedByKeys {
			dl[i] = strings.Join(d.Path, "/")
		}
		fmt.Printf("  %-12s %s\n", "Downstream:", format.Cyan(strings.Join(dl, ", ")))
	}

	if len(detail.StaleCauses) > 0 {
		fmt.Println()
		fmt.Println(format.Bold("  Stale causes:"))
		for _, c := range detail.StaleCauses {
			fmt.Printf("    %s: %s (%s)\n", format.Yellow(c.Category), c.Reason, strings.Join(c.Key.Path, "/"))
		}
	}
}

func NewInspect() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "inspect [file.parquet | asset_name]",
		Short: "Inspect a parquet file: schema, row count, populated %, Dagster info",
		Long:  "Inspect a parquet file. If no path given, lists output/ to pick. Bare names resolve to <project>/output/<name>.parquet.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveOrPick(args)
			if err != nil {
				return err
			}
			if path == "" {
				return nil
			}
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("file not found: %s", path)
			}
			if !strings.HasSuffix(strings.ToLower(path), ".parquet") {
				return fmt.Errorf("expected .parquet file, got: %s", path)
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			pf, err := parquet.OpenFile(f, fi.Size())
			if err != nil {
				return fmt.Errorf("open parquet: %w", err)
			}

			if asJSON {
				return inspectJSON(path, fi.Size(), pf)
			}

			fmt.Println(format.Bold(path))
			fmt.Println()
			fmt.Printf("  %-12s %s\n", "Size:", format.Cyan(humanBytes(fi.Size())))
			fmt.Printf("  %-12s %s\n", "Rows:", format.Cyan(fmt.Sprintf("%d", pf.NumRows())))
			fmt.Printf("  %-12s %s\n", "Row groups:", fmt.Sprintf("%d", len(pf.RowGroups())))
			fmt.Println()

			schema := pf.Schema()
			cols := schema.Columns()
			fmt.Println(format.Bold("Schema:"))
			PrintSchemaTree(FlattenSchema(schema))

			fmt.Println()
			fmt.Println(format.Bold("Populated:"))
			nulls := computeNulls(pf, cols)
			total := pf.NumRows()
			maxLeaf := 0
			leafLabels := make([]string, len(cols))
			for i, path := range cols {
				leafLabels[i] = collapseLeafPath(path)
				if len(leafLabels[i]) > maxLeaf {
					maxLeaf = len(leafLabels[i])
				}
			}
			for i, path := range cols {
				k := strings.Join(path, ".")
				populated := total - nulls[k]
				pct := 0.0
				if total > 0 {
					pct = 100.0 * float64(populated) / float64(total)
				}
				colour := format.Red
				if pct >= 50 {
					colour = format.Yellow
				}
				if pct == 100 {
					colour = format.Green
				}
				fmt.Printf("  %s  %s  %s\n",
					format.PadRight(leafLabels[i], maxLeaf),
					format.PadRight(fmt.Sprintf("%d", populated), 12),
					colour(fmt.Sprintf("%.1f%%", pct)),
				)
			}

			printDagsterSection(path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON (LLM-friendly)")
	return cmd
}

func inspectJSON(path string, size int64, pf *parquet.File) error {
	schema := pf.Schema()
	cols := schema.Columns()
	nulls := computeNulls(pf, cols)
	total := pf.NumRows()

	type colInfo struct {
		Name      string  `json:"name"`
		NullCount int64   `json:"null_count"`
		Populated int64   `json:"populated"`
		PopulatedPct float64 `json:"populated_pct"`
	}
	colsOut := make([]colInfo, len(cols))
	for i, p := range cols {
		k := strings.Join(p, ".")
		populated := total - nulls[k]
		pct := 0.0
		if total > 0 {
			pct = 100.0 * float64(populated) / float64(total)
		}
		colsOut[i] = colInfo{
			Name:         collapseLeafPath(p),
			NullCount:    nulls[k],
			Populated:    populated,
			PopulatedPct: pct,
		}
	}

	out := map[string]any{
		"path":       path,
		"size_bytes": size,
		"rows":       total,
		"row_groups": len(pf.RowGroups()),
		"schema":     FlattenSchema(schema),
		"columns":    colsOut,
	}

	base := strings.TrimSuffix(filepath.Base(path), ".parquet")
	if detail, err := client.GetAssetDetail([]string{base}); err == nil {
		out["dagster"] = dagsterSummary(detail)
	}

	PrintJSON(out)
	return nil
}

func dagsterSummary(d *client.AssetDetail) map[string]any {
	deps := make([]string, len(d.DependencyKeys))
	for i, k := range d.DependencyKeys {
		deps[i] = strings.Join(k.Path, "/")
	}
	dl := make([]string, len(d.DependedByKeys))
	for i, k := range d.DependedByKeys {
		dl[i] = strings.Join(k.Path, "/")
	}
	res := map[string]any{
		"asset":         strings.Join(d.AssetKey.Path, "/"),
		"group":         d.GroupName,
		"compute_kind":  d.ComputeKind,
		"kinds":         d.Kinds,
		"stale_status":  d.StaleStatus,
		"stale_causes":  d.StaleCauses,
		"jobs":          d.JobNames,
		"upstream":      deps,
		"downstream":    dl,
	}
	if len(d.Materializations) > 0 {
		m := d.Materializations[0]
		res["last_materialization"] = map[string]any{
			"timestamp": m.Timestamp,
			"datetime":  format.FormatTimestamp(m.Timestamp),
			"run_id":    m.RunID,
		}
	}
	return res
}

// collapseLeafPath strips Parquet list/map wrapper segments from a column path
// so null-count labels match the schema tree. e.g. constituents.list.element.Role -> constituents[].Role.
func collapseLeafPath(path []string) string {
	out := make([]string, 0, len(path))
	i := 0
	for i < len(path) {
		seg := path[i]
		lower := strings.ToLower(seg)
		if i+1 < len(path) {
			nextLower := strings.ToLower(path[i+1])
			if (nextLower == "list" || nextLower == "array") && i+2 < len(path) {
				elemLower := strings.ToLower(path[i+2])
				if elemLower == "element" || elemLower == "item" {
					out = append(out, seg+"[]")
					i += 3
					continue
				}
			}
			if nextLower == "key_value" || nextLower == "map" {
				out = append(out, seg+"{}")
				i += 2
				if i < len(path) {
					out = append(out, path[i])
					i++
				}
				continue
			}
		}
		_ = lower
		out = append(out, seg)
		i++
	}
	return strings.Join(out, ".")
}

func resolveOrPick(args []string) (string, error) {
	if len(args) == 1 {
		return project.ResolveParquet(args[0])
	}
	files, err := project.ListParquets()
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no parquet files in output/")
	}

	// Build set of known Dagster asset basenames once. Silent on Dagster failure.
	assetSet := map[string]struct{}{}
	if graph, err := client.GetAssetGraph(); err == nil {
		for _, n := range graph {
			if len(n.AssetKey.Path) > 0 {
				assetSet[n.AssetKey.Path[len(n.AssetKey.Path)-1]] = struct{}{}
			}
		}
	}

	items := make([]prompts.Item[string], len(files))
	for i, f := range files {
		base := filepath.Base(f)
		stem := strings.TrimSuffix(base, ".parquet")
		var tag string
		if _, ok := assetSet[stem]; ok {
			tag = format.Green(" [asset]")
		} else if strings.HasSuffix(stem, "_cache") {
			tag = format.Gray(" [cache]")
		} else if len(assetSet) > 0 {
			tag = format.Yellow(" [orphan]")
		}
		items[i] = prompts.Item[string]{Display: base + tag, Value: f}
	}
	picked, ok := prompts.Pick("Parquet files in output/", items)
	if !ok {
		return "", nil
	}
	return picked, nil
}

func nodeChild(n parquet.Node, name string) (parquet.Node, bool) {
	for _, f := range n.Fields() {
		if f.Name() == name {
			return f, true
		}
	}
	return nil, false
}

func nodeTypeName(n parquet.Node) string {
	if n == nil || !n.Leaf() {
		return "group"
	}
	t := n.Type()
	if t == nil {
		return "?"
	}
	if lt := t.LogicalType(); lt != nil {
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", lt)))
	}
	return strings.ToLower(t.Kind().String())
}

func computeNulls(pf *parquet.File, cols [][]string) map[string]int64 {
	out := map[string]int64{}
	for _, path := range cols {
		key := strings.Join(path, ".")
		var nulls int64
		for _, rg := range pf.RowGroups() {
			leaf, ok := rg.Schema().Lookup(path...)
			if !ok {
				continue
			}
			cc := rg.ColumnChunks()[leaf.ColumnIndex]
			pages := cc.Pages()
			for {
				page, err := pages.ReadPage()
				if err != nil || page == nil {
					break
				}
				nulls += page.NumNulls()
				parquet.Release(page)
			}
			pages.Close()
		}
		out[key] = nulls
	}
	return out
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
