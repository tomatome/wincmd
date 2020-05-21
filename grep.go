package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
//kernel32 = syscall.NewLazyDLL("kernel32.dll")
//proc     = kernel32.NewProc("SetConsoleTextAttribute")
)

type FileContent struct {
	Name  string
	Lines []Line
}

type Line struct {
	Number  int
	Content string
}

var ResultCH chan FileContent
var pattern string
var word bool
var wg sync.WaitGroup

func init() {
	ResultCH = make(chan FileContent, 10000)
}

func main() {
	flag.BoolVar(&word, "w", true, "find word")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Usage: grep content [dir]...")
		os.Exit(-1)
	}
	pattern = flag.Arg(0)
	dirs := []string{pattern, "."}
	if flag.NArg() > 1 {
		dirs = flag.Args()
	}
	//fmt.Println(dirs)
	for i, v := range dirs {
		if i == 0 {
			continue
		}
		if len(v) > 1 &&
			v[0] == '.' && v[1] != '.' {
			continue
		}
		f, _ := os.Stat(v)
		if f.IsDir() {
			FindDir(v)
			continue
		}
		if strings.HasSuffix(f.Name(), ".exe") ||
			strings.HasSuffix(f.Name(), ".tar") ||
			strings.HasSuffix(f.Name(), ".tar.gz") ||
			strings.HasSuffix(f.Name(), ".zip") ||
			strings.HasSuffix(f.Name(), ".rar") {
			continue
		}
		wg.Add(1)
		go FindFile(v)
	}
	wg.Wait()
	hasFound := false
	for {
		select {
		case file := <-ResultCH:
			for _, v := range file.Lines {
				fmt.Printf("%s(%d): ", file.Name, v.Number)
				Color(v.Content, pattern)
			}
			hasFound = true
		default:
			if !hasFound {
				fmt.Println("No found everythings")
			}
			return
		}
	}
}

func FindDir(s string) {
	dir, err := os.Open(s)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dir.Close()
	fi, err := dir.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, item := range fi {
		name := s + "/" + item.Name()

		if len(item.Name()) > 1 &&
			strings.HasPrefix(item.Name(), ".") {
			continue
		}
		if item.IsDir() {
			FindDir(name)
			continue
		}

		if word && (strings.HasSuffix(item.Name(), ".exe") ||
			strings.HasSuffix(item.Name(), ".tar") ||
			strings.HasSuffix(item.Name(), ".tar.gz") ||
			strings.HasSuffix(item.Name(), ".zip") ||
			strings.HasSuffix(item.Name(), ".rar")) {
			continue
		}

		wg.Add(1)
		go FindFile(name)
	}
}
func FindFile(file string) {
	name, err := filepath.Abs(file)
	data, err := ioutil.ReadFile(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	content := string(data)
	Lines := make([]Line, 0, 10000)
	for i, line := range strings.Split(content, "\n") {
		if len(line) <= 0 {
			continue
		}
		if strings.Contains(line, pattern) {
			Lines = append(Lines, Line{i, strings.TrimSpace(line)})
		}
	}
	if len(Lines) > 0 {
		ResultCH <- FileContent{file, Lines}
	}
	wg.Done()
}

func Color(v, a string) {
	idx := strings.Index(v, a)
	fmt.Printf("%s", v[:idx])
	//handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(12)) //12 Red light
	fmt.Printf(a)

	//handle, _, _ = proc.Call(uintptr(syscall.Stdout), uintptr(7)) //White dark
	//CloseHandle := kernel32.NewProc("CloseHandle")
	//CloseHandle.Call(handle)
	fmt.Printf("%s\n", v[idx+len(a):])
}
