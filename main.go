package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/artarts36/gomodfinder"
	cli "github.com/artarts36/singlecli"
	"log/slog"
	"os"
	"path/filepath"
)

func main() {
	app := &cli.App{
		BuildInfo: &cli.BuildInfo{
			Name:    "go-modules-action",
			Version: "v0.1.0",
		},
		Args:   []*cli.ArgDefinition{},
		Action: run,
	}

	app.RunWithGlobalArgs(context.Background())
}

func run(_ *cli.Context) error {
	modules, err := findModules()
	if err != nil {
		return fmt.Errorf("could not find modules: %w", err)
	}

	err = writeModules(modules)
	if err != nil {
		return fmt.Errorf("could not write modules: %w", err)
	}

	return nil
}

type Module struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

func findModules() ([]Module, error) {
	cwdModule, err := findModule("./")
	if err != nil {
		return nil, fmt.Errorf("failed to find module in current working directory: %w", err)
	}

	modules := []Module{cwdModule}

	pkgDirs, err := os.ReadDir("./pkg")
	if err != nil {
		if os.IsNotExist(err) {
			return modules, nil
		}

		return nil, fmt.Errorf("failed to read package directory: %w", err)
	}

	for _, pkgDir := range pkgDirs {
		if !pkgDir.IsDir() {
			continue
		}

		pkg := filepath.Join("./pkg", pkgDir.Name())

		module, mErr := findModule(pkg)
		if mErr != nil {
			return nil, fmt.Errorf("failed to find module in %q: %w", pkg, mErr)
		}

		modules = append(modules, module)
	}

	return modules, nil
}

func findModule(dir string) (Module, error) {
	mod, err := gomodfinder.Find(dir, 1)
	if err != nil {
		return Module{}, err
	}

	if mod.Module == nil {
		return Module{}, fmt.Errorf("file %q not contains module", mod.Path)
	}

	return Module{
		Name: mod.Module.Mod.Path,
		Dir:  dir,
	}, nil
}

func writeModules(modules []Module) error {
	modulesJSON, err := json.Marshal(modules)
	if err != nil {
		return fmt.Errorf("failed to marshal modules to json: %w", err)
	}

	output, ok := os.LookupEnv("GITHUB_OUTPUT")
	if !ok {
		return fmt.Errorf("GITHUB_OUTPUT not set")
	}

	outputFile, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer func(outputFile *os.File) {
		ferr := outputFile.Close()
		if ferr != nil {
			slog.With(slog.Any("err", ferr)).Error("failed to close output file")
		}
	}(outputFile)

	res := []byte(fmt.Sprintf("modules=%s", modulesJSON))

	_, err = outputFile.Write(res)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
