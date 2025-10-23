package main

import "strings"

type Buffer struct {
	Rows []string
}

func NewBuffer() *Buffer       { return &Buffer{Rows: []string{""}} }
func (b *Buffer) Join() string { return strings.Join(b.Rows, "\n") }
