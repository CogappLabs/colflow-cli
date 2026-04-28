package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/lukew-cogapp/colflow-cli/internal/project"
	"github.com/urfave/cli/v3"
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

func NewStart() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start the Dagster dev server (uv run dg dev)",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() > 0 {
				return fmt.Errorf("start takes no arguments")
			}
			return runDgDev(false)
		},
	}
}

func NewDebug() *cli.Command {
	return &cli.Command{
		Name:  "debug",
		Usage: "Start the Dagster dev server with debugpy (DAGSTER_DEBUG=1 uv run dg dev)",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() > 0 {
				return fmt.Errorf("debug takes no arguments")
			}
			return runDgDev(true)
		},
	}
}
