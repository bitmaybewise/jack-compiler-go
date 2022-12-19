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

var unaryOpsTable = map[string]string{
	"-": "neg",
	"~": "not",
}

var VarTypes = map[string]string{
	"class":    "pointer",
	"field":    "this",
	"arg":      "argument",
	"var":      "local",
	"static":   "static",
	"constant": "constant",
}

var (
	ifCounter    int
	whileCounter int
)

type Writer struct {
	out *strings.Builder
}

func (w *Writer) Output() string {
	return w.out.String()
}

func (w *Writer) WriteArithmetic(op string) error {
	val, ok := arithmeticOpsTable[op]
	if !ok {
		logger.Printf("WARNING: ignoring arithmetic symbol %q\n", op)
		return nil
	}

	_, err := w.out.WriteString(val + "\n")
	return err
}

func (w *Writer) WriteUnary(op string) error {
	val, ok := unaryOpsTable[op]
	if !ok {
		logger.Printf("WARNING: ignoring unary symbol %q\n", op)
		return nil
	}

	_, err := w.out.WriteString(val + "\n")
	return err
}

func (w *Writer) WriteKeyword(keyword string) error {
	if keyword == "true" {
		_, err := w.out.WriteString("push constant 0\n")
		if err != nil {
			return err
		}
		_, err = w.out.WriteString("not\n")
		return err
	}
	if keyword == "false" || keyword == "null" {
		_, err := w.out.WriteString("push constant 0\n")
		return err
	}
	logger.Printf("WARNING: ignoring keyword %q\n", keyword)

	return nil
}

func (w *Writer) WritePush(dest string, value any) (err error) {
	_, err = w.out.WriteString(
		fmt.Sprintf("push %s %v\n", dest, value),
	)
	return err
}

func (w *Writer) WritePop(dest string, index int) error {
	_, err := w.out.WriteString(
		fmt.Sprintf("pop %s %d\n", dest, index),
	)
	return err
}

func (w *Writer) WriteReturn() error {
	_, err := w.out.WriteString("return\n")
	if err != nil {
		return err
	}

	return nil
}

func (w *Writer) WriteSubroutine(class, subroutine string, nLocalVars int) error {
	_, err := w.out.WriteString(
		fmt.Sprintf("function %s.%s %d\n", class, subroutine, nLocalVars),
	)

	return err
}

func (w *Writer) WriteCall(subroutineType, subroutineName string, nStackVars int) error {
	_, err := w.out.WriteString(
		fmt.Sprintf("call %s.%s %d\n", subroutineType, subroutineName, nStackVars),
	)
	return err
}

func (w *Writer) WriteWhile(expressionFn func() error, statementsFn func() error) error {
	t := fmt.Sprintf("WHILE_EXP_%d", whileCounter)
	f := fmt.Sprintf("WHILE_END_%d", whileCounter)
	labelT := fmt.Sprintf("label %s\n", t)
	labelF := fmt.Sprintf("label %s\n", f)
	whileCounter++

	w.out.WriteString(labelT)
	if err := expressionFn(); err != nil { // compiled expression
		return err
	}
	w.out.WriteString("not\n")
	w.out.WriteString(fmt.Sprintf("if-goto %s\n", f))
	if err := statementsFn(); err != nil { // compiled statements
		return err
	}
	w.out.WriteString(fmt.Sprintf("goto %s\n", t))
	w.out.WriteString(labelF)

	return nil
}

func (w *Writer) WriteIf(ifFn func() error, elseFn func() error) error {
	ifFalse := fmt.Sprintf("IF_%d", ifCounter)
	ifEnd := fmt.Sprintf("IF_END_%d", ifCounter)
	labelFalse := fmt.Sprintf("label %s\n", ifFalse)
	labelEnd := fmt.Sprintf("label %s\n", ifEnd)
	ifCounter++

	w.out.WriteString("not\n")
	w.out.WriteString(fmt.Sprintf("if-goto %s\n", ifFalse))
	if err := ifFn(); err != nil { // compiled statments
		return err
	}
	w.out.WriteString(fmt.Sprintf("goto %s\n", ifEnd))
	w.out.WriteString(labelFalse)
	if err := elseFn(); err != nil { // compiled statments
		return err
	}
	w.out.WriteString(labelEnd)

	return nil
}

func New(out *strings.Builder) *Writer {
	return &Writer{out: out}
}
