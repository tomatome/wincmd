// main.go
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	te         *walk.TextEdit
	start, end int
)

func main() {
	MWindow := MainWindow{
		Title:   "CmdSSH",
		MinSize: Size{800, 600},
		Size:    Size{800, 600},
		Layout:  VBox{},
		Background: SolidColorBrush{
			Color: walk.RGB(0, 0, 0),
		},

		Children: []Widget{
			TextEdit{
				Background: SolidColorBrush{
					Color: walk.RGB(0, 0, 0),
				},
				AssignTo: &te,
				Text:     "$",
				//HScroll:   false,
				Font:      Font{Family: "Times New Roman", PointSize: 20},
				TextColor: walk.RGB(255, 255, 255),
				MaxLength: 100,
				OnTextChanged: func() {
					//te.SetFont(walk.Font{})
				},
				OnKeyDown: func(key walk.Key) {
					if key == walk.KeyReturn {
						/*te.SetTextSelection(start, end)
						cmd := te.Text()
						result := execCmd(cmd)
						te.AppendText("\r\n" + result + "\r\n$")*/
					}

				},
			},
		},
	}

	if _, err := MWindow.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execCmd(s string) string {
	cmd := exec.Command("cmd", "/C", s)
	cmd.Env = os.Environ()
	result, err := cmd.CombinedOutput()
	if err != nil {
		return err.Error()
	}

	return string(result)
}

/*type MTextEdit struct {
	*walk.TextEdit
}

func NewTextEdit() *walk.TextEdit {
	te := new(MTextEdit)
	te.SetCursor()
}
func (m *MTextEdit) Publish() {
	m.TextChanged().
}
*/
