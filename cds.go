package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"syscall"
)

func main() {
	flag.Parse()
	u, err := user.Current()
	if err != nil {
		fmt.Println("User:", err)
		os.Exit(-1)
	}
	if flag.NArg() == 0 {
		syscall.Chdir(u.HomeDir)
		err = os.Chdir(u.HomeDir)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(os.Getwd())
		os.Exit(0)
	}
	err = os.Chdir(flag.Args()[0])
	if err != nil {
		fmt.Println(err)
	}

}
