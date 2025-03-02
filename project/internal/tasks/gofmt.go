package tasks

import (
	"fmt"
	"project/pkg/workspace"
)

func GoFormatCheck() error {
	projectPath := workspace.GetProjectPath()
	output, err := workspace.GetOutput("gofmt", "-l", projectPath)
	if err != nil {
		return err
	}
	if output != "" {
		workspace.Run("gofmt", "-d", projectPath)
		return fmt.Errorf("gofmt check failed")
	}
	return nil
}

func GoFormat() error {
	projectPath := workspace.GetProjectPath()
	return workspace.Run("gofmt", "-l", "-w", projectPath)
}
