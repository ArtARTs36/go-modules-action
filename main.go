package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/artarts36/gomodfinder"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		slog.With(slog.Any("err", err)).Error("failed to get current working directory")
		os.Exit(1)
	}

	modules, err := findModules(cwd)
	if err != nil {
		slog.With(slog.Any("err", err)).Error("failed to find modules")
		os.Exit(1)
	}

	err = writeModules(modules)
	if err != nil {
		slog.With(slog.Any("err", err)).Error("failed to write modules to environment")
		os.Exit(1)
	}
}

type Module struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

func findModules(cwd string) ([]Module, error) {
	cwdModule, err := findModule(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to find module in current working directory: %w", err)
	}

	modules := []Module{cwdModule}

	pkgDirs, err := os.ReadDir(filepath.Join(cwd, "pkg"))
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

		pkg := filepath.Join(cwd, "pkg", pkgDir.Name())

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
