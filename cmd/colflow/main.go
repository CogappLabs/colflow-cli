package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lukew-cogapp/colflow-cli/internal/commands"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/urfave/cli/v3"
)

// Set by goreleaser via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Auto-load <project-root>/.env and .env.local if running inside a colflow
	// project. Existing OS env vars take precedence.
	project.LoadDotEnv(project.Cwd())

	app := &cli.Command{
		Name:            "colflow",
		Usage:           "Collection-flow CLI — Dagster pipelines, data, and operations",
		Version:         fmt.Sprintf("%s (%s, %s)", version, commit, date),
		HideHelpCommand: true,
		Commands: []*cli.Command{
			commands.NewInspect(),
			commands.NewSample(),
			commands.NewNewAsset(),
			commands.NewESCheck(),
			commands.NewStart(),
			commands.NewDebug(),
			commands.NewStatus(),
			commands.NewRuns(),
			commands.NewRun(),
			commands.NewLogs(),
			commands.NewAssets(),
			commands.NewAsset(),
			commands.NewGraph(),
			commands.NewJobs(),
			commands.NewLaunch(),
			commands.NewCancel(),
			commands.NewSensors(),
			commands.NewTail(),
			commands.NewReload(),
			commands.NewErrors(),
			commands.NewMaterialise(),
			commands.NewRecheck(),
			commands.NewStale(),
			commands.NewConfig(),
			commands.NewDiff(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
