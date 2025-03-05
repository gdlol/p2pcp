package test

import (
	"path/filepath"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func RunUnit() error {
	for _, module := range workspace.GetModules() {
		err := workspace.Run("go", "test", filepath.Join(module, "..."))
		if err != nil {
			return err
		}
	}
	return nil
}

var UnitCmd = &cobra.Command{
	Use: "unit",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunUnit()
	},
}
