package tasks

import (
	"fmt"
	"project"
)

func GoFormatCheck() error {
	projectPath := project.GetProjectPath()
	output, err := project.GetOutput("gofmt", "-l", projectPath)
	if err != nil {
		return err
	}
	if output != "" {
		project.Run("gofmt", "-d", projectPath)
		return fmt.Errorf("gofmt check failed")
	}
	return nil
}

func GoFormat() error {
	projectPath := project.GetProjectPath()
	return project.Run("gofmt", "-l", "-w", projectPath)
}
