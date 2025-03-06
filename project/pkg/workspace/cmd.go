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
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	Check(err)
}

func RunCtx(ctx context.Context, cmd string, args ...string) {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	Check(err)
}

func RunWithChdir(path string, cmd string, args ...string) {
	log.Println("chdir:", path)
	restore := Chdir(path)
	defer restore()
	Run(cmd, args...)

}

func RunCtxWithChdir(ctx context.Context, path string, cmd string, args ...string) {
	log.Println("chdir:", path)
	restore := Chdir(path)
	defer restore()
	RunCtx(ctx, cmd, args...)
}

func GetOutput(cmd string, args ...string) string {
	log.Println(cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	out, err := c.Output()
	Check(err)
	return string(out)
}
