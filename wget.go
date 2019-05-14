package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"
)

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Printf("Usage: %s Url\n", os.Args[0])
		return
	}
	args := flag.Args()
	url := args[0]
	if !strings.HasPrefix(url, "http") &&
		!strings.HasPrefix(url, "https") {
		fmt.Println("url error")
		return
	}
	s := strings.Index(url, "//") + 2
	e := strings.Index(url[s:], "/")
	host := url[s : s+e]
	fmt.Printf("Connecting to %s.....", host)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("ok\n")
	filename := filepath.Base(url)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	stat, err := f.Stat() //获取文件状态
	if err != nil {
		fmt.Println(err)
		return
	}
	f.Seek(stat.Size(), 0)

	fmt.Printf("Length: %d (%s)\n", resp.ContentLength, getLen(resp.ContentLength))
	fmt.Printf("Saving to: %s \n\n", filename)

	ln := resp.ContentLength
	bar := pb.New64(ln)
	bar.SetUnits(pb.U_BYTES_DEC)
	bar.Start()
	done := make(chan bool)
	hasDone := false
	var start int64
	go func() {
		_, err := io.Copy(f, resp.Body)
		if err != nil {
			panic(err)
		}
		done <- true
	}()
	for {
		select {
		case <-done:
			hasDone = true
		default:
		}
		stat, _ := f.Stat()
		size := stat.Size()
		//fmt.Println("size:", size, ",start:", start)
		bar.Add64(size - start)
		start = size
		if hasDone {
			break
		}
		time.Sleep(time.Millisecond)
	}
	bar.FinishPrint("\nSuccessful reception! ")

}

const (
	KB = 1024
	MB = 1024 * 1024
	GB = 1024 * 1024 * 1024
	TB = 1024 * 1024 * 1024 * 1024
)

func getLen(ln int64) (result string) {
	switch {
	case ln >= TB:
		result = fmt.Sprintf("%.02f TB", float64(ln)/TB)
	case ln >= GB:
		result = fmt.Sprintf("%.02f GB", float64(ln)/GB)
	case ln >= MB:
		result = fmt.Sprintf("%.02f MB", float64(ln)/MB)
	case ln >= KB:
		result = fmt.Sprintf("%.02f KB", float64(ln)/KB)
	default:
		result = fmt.Sprintf("%d B", ln)
	}
	return
}
