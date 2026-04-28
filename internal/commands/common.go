package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/urfave/cli/v3"
)

// CommonFlags returns the shared --url/--auth/--json set for Dagster commands.
func CommonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Usage: "Dagster base URL (default: http://127.0.0.1:3000)"},
		&cli.StringFlag{Name: "auth", Aliases: []string{"a"}, Usage: "HTTP basic auth (user:pass), or set DAGSTER_AUTH"},
		&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
	}
}

func ApplyCommon(c *cli.Command) {
	if u := c.String("url"); u != "" {
		client.SetBaseURL(u)
	}
	if a := c.String("auth"); a != "" {
		client.SetAuth(a)
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
