package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

var (
	n   int
	cmd string
)

func main() {
	flag.IntVar(&n, "n", 1, "interval time")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("%s [-n] <cmd> ")
		os.Exit(-1)
	}

	for _, args := range flag.Args() {
		cmd += args
	}
	var buf bytes.Buffer
	cmds := strings.Split(cmd, ",")
	startTime := time.Now()
	app := tview.NewApplication()
	viewer := tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetTextColor(tcell.ColorDefault)
	viewer.SetBackgroundColor(tcell.ColorDefault)
	elapsed := tview.NewTextView().SetTextColor(tcell.ColorBlack).SetTextAlign(tview.AlignRight).SetText(startTime.Format("15:04:05 2006/1/2"))
	elapsed.SetBackgroundColor(tcell.ColorRed)
	title := tview.NewTextView().SetTextColor(tcell.ColorBlack).SetText(fmt.Sprintf("Every %ds: <%s>", n, cmd))
	title.SetBackgroundColor(tcell.ColorRed)
	statusBar := tview.NewFlex().AddItem(title, 0, 1, false).AddItem(elapsed, 18, 1, false)
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	flex.AddItem(statusBar, 1, 1, true)
	flex.AddItem(viewer, 0, 1, false)
	app.SetRoot(flex, true)
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Println(err)
	}
	screen.Init()
	app.SetScreen(screen)
	go func() {
		for {
			buf.Reset()
			c := ""
			for i, a := range cmds {
				cmd := exec.Command("cmd", append([]string{"/c"}, strings.Fields(a)...)...)
				out, err := cmd.CombinedOutput()
				if err != nil {
					c = fmt.Sprintf("%s%s:%s", c, string(out), err.Error())
				} else {
					c = fmt.Sprintf("%s%s", c, string(out))
				}

				if len(cmds) > 1 && i != len(cmds)-1 {
					c = fmt.Sprintf("%s\n----------------------------------------------------------\n", c)
				}
			}
			buf.WriteString("\n" + c)

			app.QueueUpdateDraw(func() {
				screen.Clear()
				viewer.SetText(tview.TranslateANSI(buf.String()))
				elapsed.SetText(fmt.Sprintf("%v", time.Now().Format("15:04:05 2006/1/2")))
			})
			time.Sleep(time.Duration(n) * time.Second)
		}
	}()

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
