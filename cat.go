package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	proc     = kernel32.NewProc("SetConsoleTextAttribute")
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Usage: ", os.Args[0], " file[file..]")
		os.Exit(-1)
	}
	dirPth, _ := os.Getwd()

	for i, v := range flag.Args() {
		if i != 0 {
			fmt.Println("------------------------------------------------------------------")
		}
		if strings.HasPrefix(v, "/") || strings.Index(v, ":") == 1 {
			print(v)
			continue
		}
		f := filepath.Join(dirPth, v)
		print(f)
	}
}

func print(file string) {
	ColorPrintln(file+":", file+":")

	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
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
