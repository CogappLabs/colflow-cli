package project

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Info struct {
	Root        string // project root (dir containing pyproject.toml)
	Name        string // project name from [project].name
	PackageName string // python package name (underscored)
	SrcDir      string // <root>/src/<package>
	AssetsDir   string // <root>/src/<package>/defs/assets
}

var nameRe = regexp.MustCompile(`(?m)^name\s*=\s*"([^"]+)"`)

func Detect(start string) (*Info, error) {
	dir := start
	for {
		py := filepath.Join(dir, "pyproject.toml")
		if _, err := os.Stat(py); err == nil {
			return parse(py, dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, errors.New("no pyproject.toml found in any parent directory")
		}
		dir = parent
	}
}

func parse(pyPath, root string) (*Info, error) {
	b, err := os.ReadFile(pyPath)
	if err != nil {
		return nil, err
	}
	m := nameRe.FindSubmatch(b)
	if len(m) < 2 {
		return nil, errors.New("could not find project name in pyproject.toml")
	}
	name := string(m[1])
	pkg := strings.ReplaceAll(name, "-", "_")
	srcDir := filepath.Join(root, "src", pkg)
	if _, err := os.Stat(srcDir); err != nil {
		return nil, errors.New("expected src/" + pkg + " not found")
	}
	return &Info{
		Root:        root,
		Name:        name,
		PackageName: pkg,
		SrcDir:      srcDir,
		AssetsDir:   filepath.Join(srcDir, "defs", "assets"),
	}, nil
}

func Cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}
