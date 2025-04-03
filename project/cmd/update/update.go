package update

import (
	"os"
	"path/filepath"
	"project/cmd/sync"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
)

func Run() {
	workspace.Run("pnpm", "update", "--recursive", "--latest")

	for _, module := range workspace.GetModules() {
		modFile, err := os.ReadFile(filepath.Join(module, "go.mod"))
		workspace.Check(err)
		sumFile, err := os.ReadFile(filepath.Join(module, "go.sum"))
		workspace.Check(err)
		func() {
			defer func() {
				if r := recover(); r != nil {
					os.WriteFile(filepath.Join(module, "go.mod"), modFile, 0644)
					os.WriteFile(filepath.Join(module, "go.sum"), sumFile, 0644)
				}
			}()

			mod, err := modfile.Parse("go.mod", modFile, nil)
			workspace.Check(err)

			err = os.Remove(filepath.Join(module, "go.mod"))
			workspace.Check(err)
			err = os.Remove(filepath.Join(module, "go.sum"))
			workspace.Check(err)
			workspace.RunWithChdir(module, "go", "mod", "init", mod.Module.Mod.Path)

			packages := make([]string, 0, len(mod.Require))
			for _, require := range mod.Require {
				if require.Indirect {
					continue
				}
				packages = append(packages, require.Mod.Path)
			}
			workspace.RunWithChdir(module, "go", append([]string{"get"}, packages...)...)
		}()
	}

	sync.Run()
}

var UpdateCmd = &cobra.Command{
	Use: "update",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}
