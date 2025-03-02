package workspace

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

func Run(cmd string, args ...string) error {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func RunWithChdir(path string, cmd string, args ...string) error {
	log.Println("chdir:", path)
	restore := Chdir(path)
	defer restore()
	return Run(cmd, args...)
}

func GetOutput(cmd string, args ...string) (string, error) {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	out, err := c.Output()
	return string(out), err
}
