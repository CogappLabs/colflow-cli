package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads .env and .env.local from the project root if a pyproject.toml
// is found by walking up from start. Existing process env vars are NOT overridden.
// Silent on missing files. Returns the project Info if detected, else nil.
func LoadDotEnv(start string) *Info {
	info, err := Detect(start)
	if err != nil {
		return nil
	}
	for _, name := range []string{".env", ".env.local"} {
		path := filepath.Join(info.Root, name)
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path) // godotenv.Load does not override existing env vars
		}
	}
	return info
}

type Info struct {
	Root        string // project root (dir containing pyproject.toml)
	Name        string // project name from [project].name
	PackageName string // python package name (underscored)
	SrcDir      string // <root>/src/<package>
	AssetsDir   string // <root>/src/<package>/defs/assets
	OutputDir   string // <root>/output
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
		OutputDir:   filepath.Join(root, "output"),
	}, nil
}

// ResolveParquet finds a parquet file. If arg is empty, returns "" (caller should
// list output/). Tries: arg as-is, <output>/arg, <output>/arg.parquet.
func ResolveParquet(arg string) (string, error) {
	if arg == "" {
		return "", errors.New("no file specified")
	}
	if _, err := os.Stat(arg); err == nil {
		return arg, nil
	}
	info, err := Detect(Cwd())
	if err != nil {
		return "", fmt.Errorf("file not found and no project root: %s", arg)
	}
	candidates := []string{
		filepath.Join(info.OutputDir, arg),
		filepath.Join(info.OutputDir, arg+".parquet"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("file not found: tried %s, %s", arg, strings.Join(candidates, ", "))
}

// ListParquets returns parquet files in <root>/output, sorted.
func ListParquets() ([]string, error) {
	info, err := Detect(Cwd())
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(info.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("output dir not found: %s", info.OutputDir)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".parquet") {
			out = append(out, filepath.Join(info.OutputDir, e.Name()))
		}
	}
	return out, nil
}

func Cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}
