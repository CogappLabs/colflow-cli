package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/spf13/cobra"
)

type CommonFlags struct {
	URL  string
	Auth string
	JSON bool
}

func AddCommon(cmd *cobra.Command, c *CommonFlags) {
	cmd.Flags().StringVarP(&c.URL, "url", "u", "", "Dagster base URL (default: http://127.0.0.1:3000)")
	cmd.Flags().StringVarP(&c.Auth, "auth", "a", "", "HTTP basic auth (user:pass), or set DAGSTER_AUTH")
	cmd.Flags().BoolVar(&c.JSON, "json", false, "Output as JSON")
}

func (c *CommonFlags) Apply() {
	if c.URL != "" {
		client.SetBaseURL(c.URL)
	}
	if c.Auth != "" {
		client.SetAuth(c.Auth)
	}
}

func PrintJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func Die(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
