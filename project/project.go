package project

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Run(cmd string, args ...string) error {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func GetOutput(cmd string, args ...string) (string, error) {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	out, err := c.Output()
	return string(out), err
}

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

	projectPath, err := filepath.Abs(filepath.Join(filepath.Dir(file), ".."))
	Check(err)
	return projectPath
}

func ResetDir(path string) {
	err := os.RemoveAll(path)
	Check(err)
	err = os.MkdirAll(path, 0775)
	Check(err)
}

func GetTestDataPath() string {
	projectPath := GetProjectPath()
	return filepath.Join(projectPath, "test", "testdata")
}
