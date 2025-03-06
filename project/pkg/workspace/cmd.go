package workspace

import (
	"context"
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

func RunCtx(ctx context.Context, cmd string, args ...string) error {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.CommandContext(ctx, cmd, args...)
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

func RunCtxWithChdir(ctx context.Context, path string, cmd string, args ...string) error {
	log.Println("chdir:", path)
	restore := Chdir(path)
	defer restore()
	return RunCtx(ctx, cmd, args...)
}

func GetOutput(cmd string, args ...string) (string, error) {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	out, err := c.Output()
	return string(out), err
}
