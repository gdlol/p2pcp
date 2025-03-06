package tasks

import (
	"project/pkg/workspace"
)

func GoFormatCheck() {
	projectPath := workspace.GetProjectPath()
	output := workspace.GetOutput("gofmt", "-l", projectPath)

	if output != "" {
		workspace.Run("gofmt", "-d", projectPath)
		panic("gofmt check failed")
	}
}

func GoFormat() {
	projectPath := workspace.GetProjectPath()
	workspace.Run("gofmt", "-l", "-w", projectPath)
}
