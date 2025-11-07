package restore

import (
	"fmt"
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"
	"strings"
)

func GetVendorModules() []string {
	return []string{
		"github.com/libp2p/go-libp2p@v0.45.0",
	}
}

func vendor() {
	projectPath := workspace.GetProjectPath()

	// Delete and recreate vendor directory
	vendorPath := filepath.Join(projectPath, ".local", "vendor")
	workspace.ResetDir(vendorPath)
	setWritable := func() {
		workspace.Run("chmod", "--quiet", "--recursive", "u+w", vendorPath)
	}
	setWritable()
	defer setWritable()

	goModCache := workspace.GetOutput("go", "env", "GOMODCACHE")
	goModCache = strings.TrimSpace(goModCache)

	tempProjectName := fmt.Sprintf("%s-vendor", project.Name)
	tempPath := filepath.Join(os.TempDir(), tempProjectName)
	workspace.ResetDir(tempPath)
	workspace.RunWithChdir(tempPath, "go", "mod", "init", tempProjectName)

	for _, module := range GetVendorModules() {
		workspace.RunWithChdir(tempPath, "go", "get", module)
		sourcePath := filepath.Join(goModCache, module)
		targetPath := filepath.Join(vendorPath, strings.SplitN(module, "@", 2)[0])
		workspace.Run("mkdir", "--parents", targetPath)
		workspace.RunWithChdir(sourcePath, "cp", "--recursive", ".", targetPath)
	}
}

// https://github.com/libp2p/go-libp2p/issues/3415
func applyPatches() {
	projectPath := workspace.GetProjectPath()
	vendorPath := filepath.Join(projectPath, ".local", "vendor")

	mdnsPatch := filepath.Join(projectPath, "patch/mdns.patch")
	mdnsTarget := filepath.Join(vendorPath, "github.com/libp2p/go-libp2p/p2p/discovery/mdns/mdns.go")
	workspace.Run("patch", "--unified", mdnsTarget, mdnsPatch)
}
