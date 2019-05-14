package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Usage:", os.Args[0], "command")
		os.Exit(-1)
	}
	args := flag.Args()
	b := time.Now()
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out) + ":" + err.Error())
		return
	}
	fmt.Println(string(out))
	fmt.Println("\nreal: ", time.Now().Sub(b))
}
