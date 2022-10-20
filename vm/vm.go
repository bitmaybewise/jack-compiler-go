package vm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

func Output(compiled *tokenizer.NestedToken, out *strings.Builder) {
	PrintAST(compiled)
	translate(compiled, out)
}

func PrintAST(compiled *tokenizer.NestedToken) {
	fmt.Println("printing AST...")

	var line int
	var fn func(*tokenizer.NestedToken, int)
	fn = func(token *tokenizer.NestedToken, tabs int) {
		fmt.Printf("%d", line)
		line++

		for i := 0; i < tabs; i++ {
			fmt.Print("\t")
		}
		fmt.Printf("%+v\n", token)

		for _, child := range token.Children() {
			fn(child, tabs+1)
		}
	}
	fn(compiled, 1)
}

func translate(token *tokenizer.NestedToken, out *strings.Builder) {
	// fmt.Printf("%+v\n", token)

	if token.Kind == "function" {
		function(token, out)
	}
	if token.Kind == "subroutineCall" {
		subroutineCall(token, out)
		if token.Parent.Parent.Kind == "do" {
			pop("temp", 0, out)
		}
	}
	if token.Kind == "return" {
		returnCall(token, out)
	}
	if token.Kind == "var" {
		n, _ := strconv.Atoi(token.Token.Raw)
		pop("local", n, out)
	}
	if token.Token.Type == tokenizer.INT_CONST {
		push("constant", token.Token.Raw, out)
	}
	if token.Token.Type == tokenizer.SYMBOL {
		symbol(token, out)
	}

	for _, child := range token.Children() {
		translate(child, out)
	}
}

func push(dest string, value any, out *strings.Builder) {
	cmd := fmt.Sprintf("push %s %s\n", dest, value)
	out.WriteString(cmd)
}

func pop(dest string, index int, out *strings.Builder) {
	cmd := fmt.Sprintf("pop %s %d\n", dest, index)
	out.WriteString(cmd)
}

func symbol(token *tokenizer.NestedToken, out *strings.Builder) {
	op := map[string]string{
		"+": "add",
		"-": "sub",
		"*": "call Math.multiply 2",
		"/": "call Math.divide 2",
	}
	unaryOp := map[string]string{
		"-": "neg",
	}
	if val, ok := unaryOp[token.Token.Raw]; token.Kind == "unary" && ok {
		out.WriteString(val + "\n")
		return
	}
	if val, ok := op[token.Token.Raw]; ok {
		out.WriteString(val + "\n")
		return
	}
	fmt.Printf("WARNING: ignoring symbol %q\n", token.Token)
}

func function(token *tokenizer.NestedToken, out *strings.Builder) {
	cmd := fmt.Sprintf("function %s.%s 0\n", token.Parent.Token.Raw, token.Token.Raw)
	out.WriteString(cmd)
}

func subroutineCall(token *tokenizer.NestedToken, out *strings.Builder) {
	cmd := fmt.Sprintf("call %s %d\n", token.Token.Raw, len(token.Parent.Children())-1)
	out.WriteString(cmd)
}

func returnCall(token *tokenizer.NestedToken, out *strings.Builder) {
	subroutineType := token.Parent.Parent.Parent.Kind
	if subroutineType == "void" {
		push("constant", 0, out)
	}
	out.WriteString("return\n")
}
