package tasks

import "project/pkg/workspace"

func CSpell() {
	projectPath := workspace.GetProjectPath()
	workspace.Run(projectPath, "pnpm", "cspell", projectPath)
}
