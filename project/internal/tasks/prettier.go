package tasks

import "project/pkg/workspace"

func PrettierCheck() error {
	projectPath := workspace.GetProjectPath()
	return workspace.Run("pnpm", "prettier", "--check", projectPath)
}

func PrettierFormat() error {
	projectPath := workspace.GetProjectPath()
	return workspace.Run("pnpm", "prettier", "--write", projectPath)
}
