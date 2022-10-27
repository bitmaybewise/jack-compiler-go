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
		push("local", n, out)
	}
	if token.Kind == "arg" && token.Parent.Kind == "let" {
		n, _ := strconv.Atoi(token.Token.Raw)
		pop("argument", n, out)
	} else if token.Kind == "arg" {
		n, _ := strconv.Atoi(token.Token.Raw)
		push("argument", n, out)
	}
	if token.Kind == "while" {
		expression, statements := token.Children()[0], token.Children()[1]
		while(
			token,
			func() { translate(expression, out) },
			func() { translate(statements, out) },
			out,
		)
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
	cmd := fmt.Sprintf("push %s %v\n", dest, value)
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
		">": "gt",
		"<": "lt",
		"*": "call Math.multiply 2",
		"/": "call Math.divide 2",
	}
	unaryOp := map[string]string{
		"-": "neg",
		"~": "not",
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
	var nVars int
	cmd := fmt.Sprintf("function %s.%s %d\n", token.Parent.Token.Raw, token.Token.Raw, nVars)
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

var whileCounter = 0

func while(token *tokenizer.NestedToken, expressionFn, statementsFn func(), out *strings.Builder) {
	t := fmt.Sprintf("WHILE_EXP_%d", whileCounter)
	f := fmt.Sprintf("WHILE_END_%d", whileCounter)
	labelT := fmt.Sprintf("label %s\n", t)
	labelF := fmt.Sprintf("label %s\n", f)

	out.WriteString(labelT)
	expressionFn() // compiled expression
	out.WriteString("not\n")
	out.WriteString(fmt.Sprintf("if-goto %s\n", f))
	statementsFn() // compiled statements
	out.WriteString(fmt.Sprintf("goto %s\n", t))
	out.WriteString(labelF)

	whileCounter++
}
