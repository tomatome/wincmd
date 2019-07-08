package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Usage: %s file", os.Args[0])
		os.Exit(-1)
	}

	err := os.RemoveAll(flag.Args()[0])
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("successfully")
	}
}
