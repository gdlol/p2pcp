package workspace

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Run(cmd string, args ...string) {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	Check(err)
}

func RunCtx(ctx context.Context, cmd string, args ...string) {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	Check(err)
}

func RunWithChdir(path string, cmd string, args ...string) {
	log.Println("chdir:", path)
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	c.Dir = path
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	Check(err)
}

func RunCtxWithChdir(ctx context.Context, path string, cmd string, args ...string) {
	log.Println("chdir:", path)
	log.Println(cmd, strings.Join(args, " "))
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = path
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	Check(err)
}

func GetOutput(cmd string, args ...string) string {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	out, err := c.Output()
	Check(err)
	return string(out)
}
