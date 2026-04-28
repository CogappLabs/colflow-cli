package commands

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/spf13/cobra"
)

type esClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

type esIndex struct {
	Health   string `json:"health"`
	Status   string `json:"status"`
	Index    string `json:"index"`
	UUID     string `json:"uuid"`
	Pri      string `json:"pri"`
	Rep      string `json:"rep"`
	DocsCount string `json:"docs.count"`
	StoreSize string `json:"store.size"`
}

func esURL(flag string) string {
	if flag != "" {
		return strings.TrimRight(flag, "/")
	}
	if v := os.Getenv("ELASTICSEARCH_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:9200"
}

func esAuth(flag string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv("ELASTICSEARCH_API_KEY")
}

func esClient(insecure bool) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &http.Client{Transport: tr, Timeout: 15 * time.Second}
}

func esGet(url, apiKey string, insecure bool, out any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+apiKey)
	}
	req.Header.Set("Accept", "application/json")
	res, err := esClient(insecure).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return fmt.Errorf("ES %d %s: %s", res.StatusCode, res.Status, string(body))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func colourESStatus(status string) string {
	switch strings.ToLower(status) {
	case "green":
		return format.Green(status)
	case "yellow":
		return format.Yellow(status)
	case "red":
		return format.Red(status)
	}
	return format.Gray(status)
}

func NewESCheck() *cobra.Command {
	var url, apiKey string
	var insecure, asJSON, withIndices bool
	cmd := &cobra.Command{
		Use:   "es-check [index]",
		Short: "Test Elasticsearch connection; optionally check a specific index",
		Long:  "Hits ES /_cluster/health. Reads ELASTICSEARCH_URL and ELASTICSEARCH_API_KEY env vars. With an index argument, also reports that index's health + doc count. With --indices, lists all indices.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base := esURL(url)
			key := esAuth(apiKey)
			indexArg := ""
			if len(args) == 1 {
				indexArg = args[0]
			}

			var health esClusterHealth
			if err := esGet(base+"/_cluster/health", key, insecure, &health); err != nil {
				if asJSON {
					PrintJSON(map[string]any{"ok": false, "url": base, "error": err.Error()})
					return nil
				}
				fmt.Println(format.Red("ES connection failed:"), err)
				return err
			}

			var indices []esIndex
			if withIndices {
				if err := esGet(base+"/_cat/indices?format=json&bytes=b", key, insecure, &indices); err != nil {
					if !asJSON {
						fmt.Println(format.Yellow("indices fetch failed:"), err)
					}
				}
				sort.Slice(indices, func(i, j int) bool { return indices[i].Index < indices[j].Index })
			}

			var indexInfo *esIndex
			var indexErr error
			if indexArg != "" {
				var rows []esIndex
				indexErr = esGet(base+"/_cat/indices/"+indexArg+"?format=json&bytes=b", key, insecure, &rows)
				if indexErr == nil && len(rows) > 0 {
					indexInfo = &rows[0]
				}
			}

			if asJSON {
				out := map[string]any{
					"ok":      true,
					"url":     base,
					"health":  health,
					"indices": indices,
				}
				if indexArg != "" {
					ix := map[string]any{"name": indexArg, "exists": indexInfo != nil}
					if indexErr != nil {
						ix["error"] = indexErr.Error()
					}
					if indexInfo != nil {
						ix["health"] = indexInfo.Health
						ix["status"] = indexInfo.Status
						ix["docs_count"] = indexInfo.DocsCount
						ix["store_size"] = indexInfo.StoreSize
					}
					out["index"] = ix
				}
				PrintJSON(out)
				return nil
			}

			fmt.Println(format.Bold("Elasticsearch:"), base)
			fmt.Println()
			fmt.Printf("  %-14s %s\n", "Cluster:", health.ClusterName)
			fmt.Printf("  %-14s %s\n", "Status:", colourESStatus(health.Status))
			fmt.Printf("  %-14s %d (data: %d)\n", "Nodes:", health.NumberOfNodes, health.NumberOfDataNodes)
			fmt.Printf("  %-14s %d active / %d primary\n", "Shards:", health.ActiveShards, health.ActivePrimaryShards)
			if health.UnassignedShards > 0 {
				fmt.Printf("  %-14s %s\n", "Unassigned:", format.Red(fmt.Sprintf("%d", health.UnassignedShards)))
			}
			fmt.Printf("  %-14s %.1f%%\n", "Active %:", health.ActiveShardsPercentAsNumber)

			if indexArg != "" {
				fmt.Println()
				fmt.Println(format.Bold("Index:"), indexArg)
				if indexInfo == nil {
					fmt.Println("  ", format.Red("not found"))
					if indexErr != nil {
						fmt.Println(format.Gray("  "+indexErr.Error()))
					}
				} else {
					fmt.Printf("  %-14s %s\n", "Health:", colourESStatus(indexInfo.Health))
					fmt.Printf("  %-14s %s\n", "Docs:", indexInfo.DocsCount)
					fmt.Printf("  %-14s %s\n", "Size:", humanBytesStr(indexInfo.StoreSize))
				}
			}

			if withIndices && len(indices) > 0 {
				fmt.Println()
				fmt.Println(format.Bold("Indices:"))
				maxName := 0
				for _, ix := range indices {
					if len(ix.Index) > maxName {
						maxName = len(ix.Index)
					}
				}
				for _, ix := range indices {
					if strings.HasPrefix(ix.Index, ".") {
						continue
					}
					fmt.Printf("  %s  %s  %s docs  %s\n",
						format.PadRight(ix.Index, maxName),
						colourESStatus(ix.Health),
						format.PadRight(ix.DocsCount, 10),
						format.Gray(humanBytesStr(ix.StoreSize)),
					)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "ES base URL (default: $ELASTICSEARCH_URL or http://localhost:9200)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "ES API key (default: $ELASTICSEARCH_API_KEY)")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS verification")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&withIndices, "indices", false, "List indices via /_cat/indices")
	return cmd
}

// humanBytesStr passes through ES /_cat/indices store.size when it's already
// human-formatted (e.g. "12.3mb"), or formats raw bytes when bytes=b returns digits.
func humanBytesStr(s string) string {
	if s == "" {
		return ""
	}
	// If ES sent raw bytes (digits only), format.
	allDigits := true
	for _, r := range s {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if !allDigits {
		return s
	}
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return humanBytes(n)
}
