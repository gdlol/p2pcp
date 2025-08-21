package main

import (
	"fmt"
	"os"
	"test/cmd"

	_ "google.golang.org/genproto/protobuf/api"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
