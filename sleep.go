package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("You can try 'sleep 100'.")
		os.Exit(-1)
	}

	args := os.Args[1]
	t, e := strconv.Atoi(args)
	if e != nil {
		fmt.Println("You can try 'sleep 100'.")
		os.Exit(-1)
	}

	time.Sleep(time.Duration(t) * time.Second)
}
