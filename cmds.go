package main

import (
	"fmt"
	"strings"
)

type CommandFunc func(e *Editor, args []string) bool

// true -> will close editor
// false -> won't
func (e *Editor) execCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		e.SetStatus("no command")
		return false
	}

	args := strings.Fields(cmd)
	name := args[0]
	handler, ok := e.commands()[name]
	if !ok {
		e.SetStatus("unknown command: " + name)
		return false
	}

	return handler(e, args[1:])
}

func (e *Editor) commands() map[string]CommandFunc {
	return map[string]CommandFunc{
		"q":   cmdQuit,
		"qq":  cmdForceQuit,
		"w":   cmdWrite,
		"wq":  cmdWriteQuit,
		"stx": cmdSyntax,
	}
}

func cmdQuit(e *Editor, _ []string) bool {
	if e.Dirty {
		e.SetStatus("unsaved changes; save with ':w' or force quit with ':qq'")
		return false
	}
	return true
}

func cmdForceQuit(_ *Editor, _ []string) bool {
	return true
}

func cmdWrite(e *Editor, args []string) bool {
	if len(args) > 0 {
		e.Filename = args[0]
	}
	if e.Filename == "" {
		e.SetStatus("no filename specified")
		return false
	}
	if err := e.Save(); err != nil {
		e.SetStatus("save failed: " + err.Error())
		return false
	}
	e.SetStatus(fmt.Sprintf("wrote %s", e.Filename))
	return false
}

func cmdWriteQuit(e *Editor, args []string) bool {
	cmdWrite(e, args)
	return true
}

func cmdSyntax(e *Editor, args []string) bool {
	if len(args) == 0 {
		e.SetStatus("usage: :stx <path>")
		return false
	}
	// todo: parse and set syntax
	e.SetStatus("todo: load syntax from " + args[0])
	return false
}
