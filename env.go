package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	proc     = kernel32.NewProc("SetConsoleTextAttribute")
)

func main() {
	flag.Parse()
	n := flag.NArg()
	for _, v := range os.Environ() {
		if n > 0 {
			for _, a := range flag.Args() {
				if strings.Contains(v, a) {
					ColorPrintln(v, a)
				}
			}
		} else {
			fmt.Println(v)
		}

	}
}
func ColorPrintln(v, a string) {
	idx := strings.Index(v, a)
	fmt.Printf("%s", v[:idx])
	handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(12)) //12 Red light
	fmt.Printf(a)

	handle, _, _ = proc.Call(uintptr(syscall.Stdout), uintptr(7)) //White dark
	CloseHandle := kernel32.NewProc("CloseHandle")
	CloseHandle.Call(handle)
	fmt.Printf("%s\n", v[idx+len(a):])
}
