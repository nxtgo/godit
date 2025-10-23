package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 1 {
		fmt.Println("godit :D")
	}
	e, err := NewEditor()
	if err != nil {
		fmt.Println("failed to init godit:", err)
		os.Exit(1)
	}
	defer e.Close()
	if len(os.Args) >= 2 {
		_ = e.OpenFile(os.Args[1])
	}
	if err := e.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
