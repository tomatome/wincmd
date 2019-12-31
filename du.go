// du.go
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var long bool

type Detail struct {
	path string
	size int64
}

func main() {
	flag.BoolVar(&long, "l", false, "show detail info")
	flag.Parse()

	dirPth, _ := os.Getwd()
	args := flag.Args()
	if flag.NArg() == 0 {
		infos, _ := ioutil.ReadDir(dirPth)
		for _, v := range infos {
			args = append(args, v.Name())
		}
	}
	var scene sync.Map

	var wg sync.WaitGroup
	for _, v := range args {
		wg.Add(1)
		path := filepath.Join(dirPth, v)
		go func(name string) {
			f, _ := os.Stat(name)
			s := f.Size()
			if f.IsDir() {
				s = ReadDir(name)
			}
			scene.Store(name, s)
			fmt.Printf("%-8s  %s\n", Size(s), name)
			wg.Done()
		}(path)
	}

	wg.Wait()
	if flag.NArg() == 0 {
		var s int64 = 0
		scene.Range(func(k, v interface{}) bool {
			s += v.(int64)
			return true
		})
		fmt.Printf("%-8s  %s\n", Size(s), dirPth)
	}
}

func ReadDir(path string) int64 {
	infos, _ := ioutil.ReadDir(path)
	var ret int64
	//RCH := make(chan<- Detail)
	for _, v := range infos {
		s := v.Size()
		path := filepath.Join(path, v.Name())
		if v.IsDir() {
			s = ReadDir(path)
		}
		if long {
			fmt.Printf("%-8s  %s\n", Size(s), path)
		}

		ret += s
	}

	return ret
}

func Size(s int64) string {
	c := "B"
	f := float64(s)
	if f < 1024 {
		return fmt.Sprintf("%.f%s", f, c)
	}

	for _, l := range []string{"K", "M", "G", "T"} {
		if f < 1024 {
			break
		}
		f /= 1024
		c = l
	}

	return fmt.Sprintf("%.2f%s", f, c)
}
