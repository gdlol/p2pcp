package build

import (
	"fmt"
	"os"
	"p2pcp/cmd"
	"p2pcp/cmd/receive"
	"p2pcp/cmd/send"
	"path/filepath"
	"project/pkg/workspace"
	"strings"

	"github.com/spf13/cobra"
)

const template = `
# Usage

- [p2pcp](#p2pcp)
  - [p2pcp send](#p2pcp-send)
  - [p2pcp receive](#p2pcp-receive)

## |p2pcp|

|||
%s
|||

## |p2pcp send|

|||
%s
|||

## |p2pcp receive|

|||
%s
|||
`

func generateDocs() error {
	projectPath := workspace.GetProjectPath()
	docsPath := filepath.Join(projectPath, "docs")
	workspace.ResetDir(docsPath)

	rootUsage := cmd.RootCmd.UsageString()
	rootUsage = strings.Trim(rootUsage, "\n")
	sendUsage := fmt.Sprintf("%s\n\n%s", send.SendCmd.Short, send.SendCmd.UsageString())
	sendUsage = strings.Trim(sendUsage, "\n")
	receiveUsage := fmt.Sprintf("%s\n\n%s", receive.ReceiveCmd.Short, receive.ReceiveCmd.UsageString())
	receiveUsage = strings.Trim(receiveUsage, "\n")

	template := strings.Replace(template, "|", "`", -1)
	template = strings.TrimLeft(template, "\n")
	usageContent := fmt.Sprintf(template, rootUsage, sendUsage, receiveUsage)

	usageFilePath := filepath.Join(docsPath, "Usage.md")
	err := os.WriteFile(usageFilePath, []byte(usageContent), 0644)
	return err
}

var docsCmd = &cobra.Command{
	Use: "docs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateDocs()
	},
}
