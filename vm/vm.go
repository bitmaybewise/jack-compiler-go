package vm

import (
	"fmt"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

type parent interface {
	Children() []any
}

func Output(compiled any, out *strings.Builder) {
	printAST(compiled)
	translate(compiled, out)
}

const shouldPrintAST = true

func printAST(compiled any) {
	if !shouldPrintAST {
		return
	}
	fmt.Println("printing AST...")

	var line int
	var fn func(any, int)
	fn = func(token any, tabs int) {
		fmt.Printf("%d", line)
		line++

		for i := 0; i < tabs; i++ {
			fmt.Print("\t")
		}
		fmt.Printf("%+v\n", token)

		if token, ok := token.(parent); ok {
			for _, child := range token.Children() {
				fn(child, tabs+1)
			}
		}
	}
	fn(compiled, 1)
}

func translate(token any, out *strings.Builder) {
	// fmt.Printf("%+v\n", token)

	switch token := token.(type) {
	case *tokenizer.NestedToken:
		if token.Kind == "function" {
			function(token, out)
		}
		if token.Kind == "return" {
			returnCall(token, out)
		}
	case *tokenizer.Token:
		if token.Type == tokenizer.INT_CONST {
			push(token, out)
		}
		if token.Type == tokenizer.SYMBOL {
			symbol(token, out)
		}
	default:
		fmt.Printf("WARNING: ignoring unknown token %T\n", token)
	}

	if token, ok := token.(parent); ok {
		for _, child := range token.Children() {
			translate(child, out)
		}
	}
}

func push(token *tokenizer.Token, out *strings.Builder) {
	cmd := fmt.Sprintf("push constant %s\n", token.Raw)
	out.WriteString(cmd)
}

func symbol(token *tokenizer.Token, out *strings.Builder) {
	symbols := map[string]string{
		"+": "add",
		"-": "sub",
		"*": "call Math.multiply 2",
		"/": "call Math.divide 2",
		"~": "neg",
	}
	if val, ok := symbols[token.Raw]; ok {
		out.WriteString(val + "\n")
	} else {
		fmt.Printf("WARNING: ignoring symbol %q\n", token.Raw)
	}
}

func function(token *tokenizer.NestedToken, out *strings.Builder) {
	cmd := fmt.Sprintf("function %s.%s 0\n", token.Parent.Raw, token.Raw)
	out.WriteString(cmd)
}

func returnCall(token *tokenizer.NestedToken, out *strings.Builder) {
	subroutineType := token.Parent.Parent.Parent.Type

	if subroutineType == "void" {
		out.WriteString("push constant 0\n")
	}
	out.WriteString("return\n")
}
