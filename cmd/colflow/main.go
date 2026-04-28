package main

import (
	"fmt"
	"os"

	"github.com/lukew-cogapp/colflow-cli/internal/commands"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/spf13/cobra"
)

func main() {
	// Auto-load <project-root>/.env and .env.local if running inside a colflow
	// project. Existing OS env vars take precedence.
	project.LoadDotEnv(project.Cwd())

	root := &cobra.Command{
		Use:           "colflow",
		Short:         "Collection-flow CLI — Dagster pipelines, data, and operations",
		Version:       "0.1.0",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
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
		commands.NewStale(),
		commands.NewConfig(),
		commands.NewDiff(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
