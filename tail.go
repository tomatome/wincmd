// tail.go
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/hpcloud/tail"
)

func main() {
	flag.Parse()

	tails, err := tail.TailFile(flag.Args()[0], tail.Config{
		ReOpen:    true,
		Follow:    true,
		Location:  &tail.SeekInfo{Offset: -500, Whence: 2},
		MustExist: false,
		Poll:      true,
	})
	if err != nil {
		fmt.Println("tail err:", err)
		return
	}

	var msg *tail.Line
	var ok bool
	for {
		msg, ok = <-tails.Lines
		if !ok {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		fmt.Println(msg.Text)
	}
}
