package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snowmerak/aloy/internal/model"
	"github.com/snowmerak/aloy/internal/parser"
)

// Init creates a new aloy project in the given directory.
func Init(dir string) error {
	projectName := filepath.Base(dir)

	cfg := &model.ProjectConfig{
		Project: model.ProjectMeta{
			Name:        projectName,
			Version:     "0.1.0",
			CXXStandard: 17,
		},
		Targets: map[string]model.Target{
			projectName: {
				Type:    "executable",
				Sources: []string{"src/**/*.cpp"},
				Includes: model.IncludeConfig{
					Public: []string{"include/"},
				},
			},
		},
	}

	// Create directories
	dirs := []string{
		filepath.Join(dir, "src"),
		filepath.Join(dir, "include"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create a minimal main.cpp
	mainCpp := filepath.Join(dir, "src", "main.cpp")
	if _, err := os.Stat(mainCpp); os.IsNotExist(err) {
		content := `#include <iostream>

int main() {
    std::cout << "Hello from ` + projectName + `!" << std::endl;
    return 0;
}
`
		if err := os.WriteFile(mainCpp, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create main.cpp: %w", err)
		}
	}

	// Write project.yaml
	if err := parser.SaveProject(dir, cfg); err != nil {
		return err
	}

	// Append to .gitignore
	appendGitignore(dir)

	fmt.Println("Initialized aloy project:", projectName)
	fmt.Println("  project.yaml created")
	fmt.Println("  src/ and include/ directories created")
	fmt.Println("  src/main.cpp created")
	return nil
}

func appendGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	entries := "\n# aloy\n.my_modules/\nbuild/\n"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(entries)
}
