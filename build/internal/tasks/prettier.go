package tasks

import project "build/internal"

func PrettierCheck() error {
	projectPath := project.GetProjectPath()
	return project.Run("pnpm", "prettier", "--check", projectPath)
}

func PrettierFormat() error {
	projectPath := project.GetProjectPath()
	return project.Run("pnpm", "prettier", "--write", projectPath)
}
