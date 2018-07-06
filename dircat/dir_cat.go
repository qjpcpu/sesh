package dircat

import (
	"fmt"
	"github.com/qjpcpu/sesh/gocui"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var files []string
var file_offset, window_offset int

const (
	Padding     = 1
	PrefixEmpty = "   "
	PrefixArrow = "=> "
)

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if _, err := g.SetView("center", 0, 0, maxX*4/5, maxY-Padding); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	if v, err := g.SetView("side", maxX*4/5+1, 0, maxX-Padding, maxY-Padding); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		_, height := g.Size()
		if height-2*Padding > len(files) {
			height = len(files)
		}

		for i := 0; i < height; i++ {
			name := filepath.Base(files[i])
			if i == 0 {
				fmt.Fprintln(v, PrefixArrow+name)
			} else {
				fmt.Fprintln(v, PrefixEmpty+name)
			}

		}
	}
	return nil
}
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrorQuit
}
func nexOne(g *gocui.Gui, v *gocui.View) error {
	if lv := g.View("side"); lv != nil {
		_, height := g.Size()
		height = height - 2*Padding
		length := len(files)
		file_offset = (file_offset + 1 + length) % length
		drawSide(g, lv, files, file_offset)
	}
	return nil
}
func prevOne(g *gocui.Gui, v *gocui.View) error {
	if lv := g.View("side"); lv != nil {
		_, height := g.Size()
		height = height - 2*Padding
		length := len(files)
		file_offset = (file_offset - 1 + length) % length
		drawSide(g, lv, files, file_offset)
	}
	return nil
}
func drawSide(g *gocui.Gui, side *gocui.View, filelist []string, fOffset int) {
	side.Clear()
	_, wheight := g.Size()
	wheight = wheight - 2*Padding
	if fOffset < window_offset {
		window_offset = fOffset
	} else if fOffset >= window_offset+wheight {
		window_offset = fOffset - wheight + 1
	}
	end := wheight + window_offset
	if len(filelist) < end {
		end = len(filelist)
	}
	for i, name := range filelist {
		if i < window_offset || i >= end {
			continue
		}
		bname := filepath.Base(name)
		if i == fOffset {
			fmt.Fprintln(side, PrefixArrow+bname)
		} else {
			fmt.Fprintln(side, PrefixEmpty+bname)
		}
	}
}
func Init(filelist ...string) (*DirCat, error) {
	list := []string{}
	for _, f := range filelist {
		if fi, err := os.Stat(f); err == nil && !fi.IsDir() {
			fn, _ := filepath.Abs(f)
			list = append(list, fn)
		}
	}
	files = list
	file_offset, window_offset = 0, 0
	dc := &DirCat{}
	g := gocui.NewGui()
	dc.controller = g
	if err := g.Init(); err != nil {
		return nil, err
	}
	dc.controller = g
	g.SetLayout(layout)
	if err := g.SetKeybinding("", gocui.KeyCtrlC, 0, quit); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, 0, nexOne); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlN, 0, nexOne); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, 0, prevOne); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlP, 0, prevOne); err != nil {
		return nil, err
	}
	return dc, nil
}
func drawCenter(g *gocui.Gui) {
	if v := g.View("center"); v != nil {
		_, height := g.Size()
		height = height - 2
		if text, err := TailFile(files[file_offset], height); err == nil {
			if i := strings.Index(text, "\n"); i > 0 {
				re := regexp.MustCompile("\\033[[0-9]+m")
				line := re.ReplaceAllString(text[0:i], "")
				text = line + text[i:]
			} else {
				re := regexp.MustCompile("\\033[[0-9]+m")
				text = re.ReplaceAllString(text, "")
			}
			v.Clear()
			fmt.Fprintf(v, "%s", text)
			g.Flush()
		} else {
			v.Clear()
			fmt.Fprintf(v, "------ failed to read from %s ------\n", filepath.Base(files[file_offset]))
			g.Flush()
		}
	}
}

type DirCat struct {
	controller *gocui.Gui
}

func (dc *DirCat) Stop() {
	dc.controller.UserMessage <- "quit1"
	dc.controller.UserMessage <- "quit2"
}
func (dc *DirCat) Start() {
	defer dc.controller.Close()
	go func() {
		c := time.Tick(500 * time.Millisecond)
	LoopFlush:
		for {
			select {
			case <-c:
				drawCenter(dc.controller)
			case <-dc.controller.UserMessage:
				break LoopFlush
			}
		}
	}()
	err := dc.controller.MainLoop()
	if err != nil && err != gocui.ErrorQuit {
		fmt.Println(err)
		return
	}
}
