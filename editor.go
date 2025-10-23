package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Editor struct {
	Buf        *Buffer
	CX, CY     int
	RowOff     int
	ColOff     int
	Rows, Cols int
	Filename   string
	Status     string
	StatusAt   time.Time
	Dirty      bool
	Mode       string
	CmdLine    []rune
	orig       syscallTermios
}

func NewEditor() (*Editor, error) {
	e := &Editor{Mode: "normal", Buf: NewBuffer()}
	if err := enableRaw(&e.orig); err != nil {
		return nil, err
	}
	r, c, err := winsize()
	if err != nil {
		disableRaw(&e.orig)
		return nil, err
	}
	e.Rows = r - 2
	e.Cols = c
	return e, nil
}

func (e *Editor) Close() { disableRaw(&e.orig); fmt.Print("\x1b[2J\x1b[H\x1b[?25h") }

func (e *Editor) OpenFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	e.Filename = path
	s := string(b)
	sc := bufio.NewScanner(strings.NewReader(s))
	e.Buf = NewBuffer()
	e.Buf.Rows = nil
	for sc.Scan() {
		e.Buf.Rows = append(e.Buf.Rows, sc.Text())
	}
	if len(e.Buf.Rows) == 0 {
		e.Buf.Rows = append(e.Buf.Rows, "")
	}
	e.Dirty = false
	return nil
}

func (e *Editor) Save() error {
	if e.Filename == "" {
		return fmt.Errorf("no filename")
	}
	out := strings.Join(e.Buf.Rows, "\n")
	if err := os.WriteFile(e.Filename, []byte(out), 0644); err != nil {
		return err
	}
	e.Dirty = false
	e.SetStatus(fmt.Sprintf("%d bytes written to %s", len(out), e.Filename))
	return nil
}

func (e *Editor) SetStatus(s string) { e.Status = s; e.StatusAt = time.Now() }

func (e *Editor) Run() error {
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")

	for {
		if err := e.refresh(); err != nil {
			return err
		}

		k, err := readKey()
		if err != nil {
			return err
		}

		switch e.Mode {
		case "command":
			if quit := e.handleCommandMode(k); quit {
				return nil
			}
			continue

		case "insert":
			e.handleInsertMode(k)
			continue
		}

		switch k {
		case Cfg.Keymap["Quit"]:
			if e.Dirty {
				e.SetStatus("unsaved changes. use :qq to quit or :w to save")
				continue
			}
			return nil

		case Cfg.Keymap["Save"]:
			_ = e.Save()

		case Cfg.Keymap["Command"]:
			e.Mode = "command"
			e.CmdLine = nil

		case Cfg.Keymap["Insert"]:
			e.Mode = "insert"

		case 'h', KeyArrowLeft:
			e.moveLeft()
		case 'l', KeyArrowRight:
			e.moveRight()
		case 'k', KeyArrowUp:
			e.moveUp()
		case 'j', KeyArrowDown:
			e.moveDown()
		case KeyHome:
			e.CX = 0
		case KeyEnd:
			if e.CY < len(e.Buf.Rows) {
				e.CX = len(e.Buf.Rows[e.CY])
			}
		case KeyPageUp:
			for i := 0; i < e.Rows; i++ {
				e.moveUp()
			}
		case KeyPageDown: // Command requested exit
			for i := 0; i < e.Rows; i++ {
				e.moveDown()
			}
		case KeyDel:
			e.delchar()
		}
	}
}

func (e *Editor) handleCommandMode(k int) bool {
	switch k {
	case KeyEnter:
		cmd := string(e.CmdLine)
		e.CmdLine = nil
		e.Mode = "normal"
		if e.execCommand(cmd) {
			return true
		}
	case KeyEsc:
		e.Mode = "normal"
		e.CmdLine = nil
	case KeyBackspace:
		if len(e.CmdLine) > 0 {
			e.CmdLine = e.CmdLine[:len(e.CmdLine)-1]
		}
	default:
		if k >= 32 && k <= 126 {
			e.CmdLine = append(e.CmdLine, rune(k))
		}
	}
	return false
}

func (e *Editor) handleInsertMode(k int) {
	if k == KeyEsc {
		e.Mode = "normal"
		return
	}
	if k == KeyEnter {
		e.insertNewline()
		return
	}
	if k == KeyBackspace {
		e.backspace()
		return
	}
	if k >= 32 && k <= 126 {
		e.insertChar(rune(k))
		return
	}
}

func (e *Editor) moveLeft() {
	if e.CX > 0 {
		e.CX--
	} else if e.CY > 0 {
		e.CY--
		e.CX = len(e.Buf.Rows[e.CY])
	}
	if e.CX < 0 {
		e.CX = 0
	}
	e.scroll()
}

func (e *Editor) moveRight() {
	line := ""
	if e.CY < len(e.Buf.Rows) {
		line = e.Buf.Rows[e.CY]
	}
	if e.CX < len(line) {
		e.CX++
	} else if e.CY+1 < len(e.Buf.Rows) {
		e.CY++
		e.CX = 0
	}
	e.scroll()
}

func (e *Editor) moveUp() {
	if e.CY > 0 {
		e.CY--
		if e.CX > len(e.Buf.Rows[e.CY]) {
			e.CX = len(e.Buf.Rows[e.CY])
		}
	}
	e.scroll()
}

func (e *Editor) moveDown() {
	if e.CY+1 < len(e.Buf.Rows) {
		e.CY++
		if e.CX > len(e.Buf.Rows[e.CY]) {
			e.CX = len(e.Buf.Rows[e.CY])
		}
	}
	e.scroll()
}

func (e *Editor) scroll() {
	if e.CY < e.RowOff {
		e.RowOff = e.CY
	}
	if e.CY >= e.RowOff+e.Rows {
		e.RowOff = e.CY - e.Rows + 1
	}
	if e.CX < e.ColOff {
		e.ColOff = e.CX
	}
	if e.CX >= e.ColOff+e.Cols {
		e.ColOff = e.CX - e.Cols + 1
	}
}

func (e *Editor) refresh() error {
	var b strings.Builder
	b.WriteString("\x1b[H")

	numWidth := len(fmt.Sprintf("%d", len(e.Buf.Rows))) + 1

	for y := 0; y < e.Rows; y++ {
		f := y + e.RowOff
		if f >= len(e.Buf.Rows) {
			b.WriteString("~\x1b[K\r\n")
		} else {
			lineNum := f + 1
			b.WriteString(fmt.Sprintf("\033[90m%*d\033[0m ", numWidth-1, lineNum))

			line := e.Buf.Rows[f]
			visible := ""
			if e.ColOff < len(line) {
				visible = line[e.ColOff:]
			}
			if len(visible) > e.Cols-numWidth {
				visible = visible[:e.Cols-numWidth]
			}
			b.WriteString(visible)
			b.WriteString("\x1b[K\r\n")
		}
	}

	// status line
	b.WriteString(Cfg.Theme.Invert)
	left := e.Filename
	if left == "" {
		left = "[unknown]"
	}
	if e.Dirty {
		left += " [+]"
	}
	left = left + " - " + e.Mode
	if len(left) > e.Cols {
		left = left[:e.Cols]
	}
	right := fmt.Sprintf("%d/%d", e.CY+1, len(e.Buf.Rows))
	b.WriteString(left)
	for i := len(left); i < e.Cols-len(right); i++ {
		b.WriteByte(' ')
	}
	b.WriteString(right)
	b.WriteString(Cfg.Theme.Reset + "\r\n")
	b.WriteString("\x1b[K")

	if e.Mode == "command" {
		b.WriteString(":" + string(e.CmdLine))
	} else {
		if time.Since(e.StatusAt) < Cfg.StatusTime {
			msg := e.Status
			if len(msg) > e.Cols {
				msg = msg[:e.Cols]
			}
			b.WriteString(msg)
		}
	}

	cy := e.CY - e.RowOff + 1
	cx := e.CX - e.ColOff + numWidth
	if cx < numWidth {
		cx = numWidth
	}
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH", cy, cx))
	b.WriteString("\x1b[?25h")
	_, err := os.Stdout.WriteString(b.String())
	return err
}

func (e *Editor) insertChar(r rune) {
	if e.CY >= len(e.Buf.Rows) {
		e.Buf.Rows = append(e.Buf.Rows, "")
	}
	line := e.Buf.Rows[e.CY]
	if e.CX < 0 || e.CX > len(line) {
		e.CX = len(line)
	}
	line = line[:e.CX] + string(r) + line[e.CX:]
	e.Buf.Rows[e.CY] = line
	e.CX++
	e.Dirty = true
}

func (e *Editor) backspace() {
	if e.CY == 0 && e.CX == 0 {
		return
	}
	line := e.Buf.Rows[e.CY]
	if e.CX > 0 {
		line = line[:e.CX-1] + line[e.CX:]
		e.Buf.Rows[e.CY] = line
		e.CX--
	} else {
		prev := e.Buf.Rows[e.CY-1]
		e.CX = len(prev)
		e.Buf.Rows[e.CY-1] = prev + line
		e.Buf.Rows = append(e.Buf.Rows[:e.CY], e.Buf.Rows[e.CY+1:]...)
		e.CY--
	}
	e.Dirty = true
}

func (e *Editor) delchar() {
	line := e.Buf.Rows[e.CY]
	if e.CX >= len(line) {
		if e.CY+1 < len(e.Buf.Rows) {
			e.Buf.Rows[e.CY] = line + e.Buf.Rows[e.CY+1]
			e.Buf.Rows = append(e.Buf.Rows[:e.CY+1], e.Buf.Rows[e.CY+2:]...)
			e.Dirty = true
		}
	} else {
		line = line[:e.CX] + line[e.CX+1:]
		e.Buf.Rows[e.CY] = line
		e.Dirty = true
	}
}

func (e *Editor) insertNewline() {
	line := e.Buf.Rows[e.CY]
	if e.CX == 0 {
		e.Buf.Rows = append(e.Buf.Rows[:e.CY], append([]string{""}, e.Buf.Rows[e.CY:]...)...)
	} else if e.CX >= len(line) {
		if e.CY+1 >= len(e.Buf.Rows) {
			e.Buf.Rows = append(e.Buf.Rows, "")
		} else {
			e.Buf.Rows = append(e.Buf.Rows[:e.CY+1], append([]string{""}, e.Buf.Rows[e.CY+1:]...)...)
		}
	} else {
		left, right := line[:e.CX], line[e.CX:]
		e.Buf.Rows[e.CY] = left
		if e.CY+1 >= len(e.Buf.Rows) {
			e.Buf.Rows = append(e.Buf.Rows, right)
		} else {
			e.Buf.Rows = append(e.Buf.Rows[:e.CY+1], append([]string{right}, e.Buf.Rows[e.CY+1:]...)...)
		}
	}
	e.CY++
	e.CX = 0
	e.Dirty = true
}
