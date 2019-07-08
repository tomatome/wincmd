package main

import (
	"fmt"
	"os"
)

func main() {
	d, e := os.Getwd()
	if e != nil {
		fmt.Println(e)
		os.Exit(-1)
	}
	fmt.Println("CWD:", d)
}
