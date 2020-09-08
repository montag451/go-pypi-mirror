package main

import (
	"fmt"

	"github.com/montag451/go-pypi-mirror/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
