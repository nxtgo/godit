package main

import "time"

type Config struct {
	TabWidth   int
	StatusTime time.Duration
	Theme      struct {
		Invert string
		Reset  string
	}
	Keymap map[string]int
}

var Cfg = Config{
	TabWidth:   4,
	StatusTime: 5 * time.Second,
}

func init() {
	Cfg.Theme.Invert = "\x1b[7m"
	Cfg.Theme.Reset = "\x1b[m"
	Cfg.Keymap = map[string]int{
		"Quit":    KeyQuit,
		"Save":    KeySave,
		"Insert":  'i',
		"Command": ':',
	}
}
