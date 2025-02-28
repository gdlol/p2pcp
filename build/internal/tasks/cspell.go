package tasks

import (
	project "build/internal"
)

func CSpell() error {
	projectPath := project.GetProjectPath()
	return project.Run("pnpm", "cspell", projectPath)
}
