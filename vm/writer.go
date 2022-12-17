package vm

import (
	"fmt"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/logger"
)

var arithmeticOpsTable = map[string]string{
	"+": "add",
	"-": "sub",
	"=": "eq",
	">": "gt",
	"<": "lt",
	"&": "and",
	"|": "or",
	"*": "call Math.multiply 2",
	"/": "call Math.divide 2",
}

type Writer struct {
	out *strings.Builder
}

func (b *Writer) Output() {}

func (b *Writer) WriteArithmetic(op string) error {
	val, ok := arithmeticOpsTable[op]
	if !ok {
		logger.Printf("WARNING: ignoring arithmetic symbol %q\n", op)
		return nil
	}

	_, err := b.out.WriteString(val + "\n")
	return err
}

func (b *Writer) WritePush(dest string, value any) error {
	_, err := b.out.WriteString(
		fmt.Sprintf("push %s %v\n", dest, value),
	)
	return err
}

func (b *Writer) WritePop(dest string, index int) error {
	_, err := b.out.WriteString(
		fmt.Sprintf("pop %s %d\n", dest, index),
	)
	return err
}

func (b *Writer) WriteReturn() error {
	err := b.WritePush("constant", 0)
	if err != nil {
		return err
	}
	_, err = b.out.WriteString("return\n")
	if err != nil {
		return err
	}
	return nil
}

func (b *Writer) WriteSubroutine(class, subroutine string, nLocalVars int) error {
	_, err := b.out.WriteString(
		fmt.Sprintf("function %s.%s %d\n", class, subroutine, nLocalVars),
	)

	return err
}

func (b *Writer) WriteCall(subroutineType, subroutineName string, nStackVars int) error {
	_, err := b.out.WriteString(
		fmt.Sprintf("call %s.%s %d\n", subroutineType, subroutineName, nStackVars+1),
	)
	return err
}

func New(out *strings.Builder) *Writer {
	return &Writer{out: out}
}
