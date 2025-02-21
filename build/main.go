package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func run(cmd string, args ...string) error {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) == 1 {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			log.Fatalln("runtime.Caller failed")
		}

		projectPath := filepath.Dir(filepath.Dir(file))

		err := run("go", "build",
			"-o", filepath.Join(projectPath, "bin")+string(os.PathSeparator),
			filepath.Join(projectPath, "p2pcp"))
		check(err)
	}
}
