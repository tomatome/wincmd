// ls.go
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/disiqueira/gotree"
	"github.com/tomatome/win"
)

type LS struct {
	long   bool
	tree   bool
	files  []os.FileInfo
	filter string
}

var (
	dirPth string
	ls     LS
)

func main() {
	flag.BoolVar(&ls.long, "l", false, "show files long format")
	flag.BoolVar(&ls.tree, "t", false, "show files tree format")
	flag.Parse()

	if flag.NArg() == 0 {
		dirPth, _ = os.Getwd()
		if ls.tree {
			fmt.Println(Tree(dirPth).Print())
			return
		}
		ls.getAllFiles(dirPth)
		if ls.long {
			ls.PrintLong()
		} else {
			ls.Print()
		}
		return
	}

	argv := flag.Args()
	num := len(argv)
	for i, a := range argv {
		if ls.tree {
			path, _ := filepath.Abs(a)
			fmt.Println(path, ":")
			fmt.Println(Tree(path).Print())
			if flag.NArg() > 0 && i != flag.NArg()-1 {
				fmt.Println("------------------------------------------------------")
			}
			continue
		}
		dir := a
		ls.filter = ""
		if strings.HasSuffix(dir, "*") {
			base := path.Base(a)
			ls.filter = base[:len(base)-1]
			dir = path.Dir(a)
			//fmt.Println(ls.filter)
		}
		if num > 1 {
			fmt.Println(a, " :")
		}
		ls.getAllFiles(dir)
		if ls.long {
			ls.PrintLong()
		} else {
			ls.Print()
		}

		if num > 1 && i != num-1 {
			fmt.Println("----------------------------------------------------------------")
		}
	}

}

func (l *LS) getAllFiles(dirPth string) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	l.files = dir
}

func (l LS) getNum() (int, int) {
	var r win.RECT
	h := win.GetConsoleWindow()
	win.GetClientRect(h, &r)
	w := r.Right - r.Left
	maxln := 0
	for _, f := range l.files {
		if maxln < len(f.Name()) {
			maxln = len(f.Name())
		}
	}
	//fmt.Println(w)
	num := int(w) / 2 / (6*maxln + 10)
	if num == 0 {
		num = 1
	}
	return num, maxln + 8
}

func (l LS) Print() {
	num, length := l.getNum()
	//fmt.Println(num, ":", length)
	name := ""
	i := 0
	for _, f := range l.files {
		name = f.Name()
		if l.filter != "" && !strings.HasPrefix(name, l.filter) {
			continue
		}
		if f.IsDir() {
			name += "/"
		}
		ln := length
		if (i+1)%num == 0 {
			ln = len(name)
		}
		s := fmt.Sprintf("%s-%ds", "%", ln)
		fmt.Printf(s, name)

		if (i+1)%num == 0 {
			fmt.Printf("\n")
		}
		i++
	}
}

func (l LS) PrintLong() {
	for _, f := range l.files {
		if l.filter != "" && !strings.HasPrefix(f.Name(), l.filter) {
			continue
		}
		fmt.Printf("%10s", f.Mode().String())
		//fmt.Printf(f.Sys())
		fmt.Printf("   ")
		fmt.Printf("%8s", size(f.Size()))
		fmt.Printf("   ")
		fmt.Printf("%10s", f.ModTime().Format("2006/01/02 15:04:05"))
		fmt.Printf("   ")
		fmt.Printf(f.Name())
		if f.IsDir() {
			fmt.Printf("/")
		}
		fmt.Println("")
	}
}

func size(s int64) string {
	c := "B"
	f := float64(s)
	if f < 1024 {
		return fmt.Sprintf("%.f %s", f, c)
	}

	for _, l := range []string{"K", "M", "G", "T"} {
		if f < 1024 {
			break
		}
		f /= 1024
		c = l
	}

	return fmt.Sprintf("%.2f %s", f, c)
}

func Tree(dirPth string) gotree.Tree {
	artist := gotree.New(filepath.Base(dirPth) + "/")
	infos, _ := ioutil.ReadDir(dirPth)
	for _, v := range infos {
		if v.IsDir() {
			artist.AddTree(Tree(filepath.Join(dirPth, v.Name())))
			continue
		}
		artist.Add(v.Name())
	}
	return artist
}
