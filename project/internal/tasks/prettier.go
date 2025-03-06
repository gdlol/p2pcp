package tasks

import (
	"project/pkg/workspace"
)

func PrettierCheck() {
	workspacesPath := workspace.GetWorkspacesPath()
	projectPath := workspace.GetProjectPath()
	workspace.RunWithChdir(workspacesPath, "pnpm", "prettier", "--check", projectPath)
}

func PrettierFormat() {
	workspacesPath := workspace.GetWorkspacesPath()
	projectPath := workspace.GetProjectPath()
	workspace.RunWithChdir(workspacesPath, "pnpm", "prettier", "--write", projectPath)
}
