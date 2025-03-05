package test

import (
	"github.com/spf13/cobra"
)

var TestCmd = &cobra.Command{
	Use: "test",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := RunUnit()
		if err != nil {
			return err
		}
		return RunIntegration()
	},
}

func init() {
	TestCmd.AddCommand(UnitCmd)
	TestCmd.AddCommand(IntegrationCmd)
}
