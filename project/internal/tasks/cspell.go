package tasks

import "project/pkg/workspace"

func CSpell() error {
	projectPath := workspace.GetProjectPath()
	return workspace.Run("pnpm", "cspell", projectPath)
}
