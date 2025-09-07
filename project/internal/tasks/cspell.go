package tasks

import "project/pkg/workspace"

func CSpell() {
	projectPath := workspace.GetProjectPath()
	workspacePath := workspace.GetWorkspacesPath()
	workspace.RunWithChdir(workspacePath, "pnpm", "cspell", projectPath)
}
