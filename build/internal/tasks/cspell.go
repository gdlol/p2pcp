package tasks

import "project"

func CSpell() error {
	projectPath := project.GetProjectPath()
	return project.Run("pnpm", "cspell", projectPath)
}
