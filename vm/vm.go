package vm

import (
	"strings"
)

type Writer struct {
	out *strings.Builder
}

func NewWriter(out *strings.Builder) *Writer {
	return &Writer{out}
}

func (w *Writer) Output() (string, error) {
	return w.out.String(), nil
}
