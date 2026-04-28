package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/spf13/cobra"
)

func runDgDev(debug bool) error {
	info, err := project.Detect(project.Cwd())
	if err != nil {
		return err
	}
	c := exec.Command("uv", "run", "dg", "dev")
	c.Dir = info.Root
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()
	if debug {
		c.Env = append(c.Env, "DAGSTER_DEBUG=1")
	}

	label := "dg dev"
	if debug {
		label = "dg dev (DAGSTER_DEBUG=1)"
	}
	fmt.Printf("%s %s in %s\n", format.Bold("Starting"), label, info.Root)
	return c.Run()
}

func NewStart() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the Dagster dev server (uv run dg dev)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDgDev(false)
		},
	}
}

func NewDebug() *cobra.Command {
	return &cobra.Command{
		Use:   "debug",
		Short: "Start the Dagster dev server with debugpy (DAGSTER_DEBUG=1 uv run dg dev)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDgDev(true)
		},
	}
}
