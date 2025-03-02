package workspace

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/mod/modfile"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func GetProjectPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalln("runtime.Caller failed")
	}

	projectPath, err := filepath.Abs(filepath.Join(filepath.Dir(file), "../../.."))
	Check(err)
	return projectPath
}

func GetTestDataPath() string {
	projectPath := GetProjectPath()
	return filepath.Join(projectPath, "test", "testdata")
}

func GetModules() []string {
	projectPath := GetProjectPath()
	data, err := os.ReadFile(filepath.Join(projectPath, "go.work"))
	Check(err)
	workfile, err := modfile.ParseWork("go.work", data, nil)
	Check(err)
	modules := make([]string, 0)
	for _, use := range workfile.Use {
		modules = append(modules, filepath.Join(projectPath, use.Path))
	}
	return modules
}
