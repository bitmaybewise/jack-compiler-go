package vm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

func Output(compiled *tokenizer.Token, out *strings.Builder) {
	PrintAST(compiled)
	translate(compiled, out)
}

func PrintAST(compiled *tokenizer.Token) {
	fmt.Println("printing AST...")

	var line int
	var fn func(*tokenizer.Token, int)
	fn = func(token *tokenizer.Token, tabs int) {
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

func translate(token *tokenizer.Token, out *strings.Builder) {
	// fmt.Printf("%+v\n", token)

	if token == nil {
		return
	}
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
		for _, expression := range token.Children() {
			translate(expression, out)
		}
		returnCall(token, out)
		return
	}
	if token.Kind == "varDec" {
		varDec(token, out)
	}
	if token.Kind == "var" && token.Parent.Kind == "let" {
		pop("local", token.Var.Index, out)
	}
	if token.Kind == "var" && token.Parent.Kind != "let" {
		push("local", token.Var.Index, out)
	}
	if token.Kind == "varAssignment" {
		n, _ := strconv.Atoi(token.Raw)
		pop("local", n, out)
	}
	if token.Kind == "arg" && token.Parent.Kind == "let" {
		pop("argument", token.Var.Index, out)
	}
	if token.Kind == "arg" && token.Parent.Kind != "let" {
		push("argument", token.Var.Index, out)
	}
	if token.Kind == "while" {
		children := token.Children()
		expression, statements := children[0], children[1]
		while(
			token,
			func() { translate(expression, out) },
			func() { translate(statements, out) },
			out,
		)
		return
	}
	if token.Kind == "if" {
		children := token.Children()
		exp := children[0]
		translate(exp, out)
		trueStatements := children[1]
		falseStatements := children[3]
		ifStatement(
			token,
			func() { translate(trueStatements, out) },
			func() { translate(falseStatements, out) },
			out,
		)
		return
	}
	if token.Type == tokenizer.INT_CONST {
		push("constant", token.Raw, out)
	}
	if token.Type == tokenizer.SYMBOL {
		symbol(token, out)
	}
	if token.Type == tokenizer.KEYWORD {
		keyword(token, out)
	}

	for _, child := range token.Children() {
		translate(child, out)
	}
}

func varDec(token *tokenizer.Token, out *strings.Builder) {
}

func push(dest string, value any, out *strings.Builder) {
	cmd := fmt.Sprintf("push %s %v\n", dest, value)
	out.WriteString(cmd)
}

func pop(dest string, index int, out *strings.Builder) {
	cmd := fmt.Sprintf("pop %s %d\n", dest, index)
	out.WriteString(cmd)
}

func symbol(token *tokenizer.Token, out *strings.Builder) {
	op := map[string]string{
		"+": "add",
		"-": "sub",
		"=": "eq",
		">": "gt",
		"<": "lt",
		"&": "and",
		"*": "call Math.multiply 2",
		"/": "call Math.divide 2",
	}
	unaryOp := map[string]string{
		"-": "neg",
		"~": "not",
	}
	if val, ok := unaryOp[token.Raw]; token.Kind == "unary" && ok {
		out.WriteString(val + "\n")
		return
	}
	if val, ok := op[token.Raw]; ok {
		out.WriteString(val + "\n")
		return
	}
	fmt.Printf("WARNING: ignoring symbol %q\n", token.Raw)
}

func keyword(token *tokenizer.Token, out *strings.Builder) {
	if token.Type == tokenizer.KEYWORD && token.Raw == "true" {
		out.WriteString("push constant 0\n")
		out.WriteString("not\n")
		return
	}
	if token.Type == tokenizer.KEYWORD && token.Raw == "false" {
		out.WriteString("push constant 0\n")
		return
	}
	fmt.Printf("WARNING: ignoring keyword %q, parent %q\n", token.Raw, token.Parent.Raw)
}

func function(token *tokenizer.Token, out *strings.Builder) {
	cmd := fmt.Sprintf("function %s.%s %d\n", token.Parent.Raw, token.Raw, token.NLocalVars())
	out.WriteString(cmd)
}

func subroutineCall(token *tokenizer.Token, out *strings.Builder) {
	cmd := fmt.Sprintf("call %s %d\n", token.Raw, token.NStackVars())
	out.WriteString(cmd)
}

func returnCall(token *tokenizer.Token, out *strings.Builder) {
	subroutineType := token.Parent.Parent.Parent.Kind
	if subroutineType == "void" {
		push("constant", 0, out)
	}
	out.WriteString("return\n")
}

var whileCounter = 0

func while(token *tokenizer.Token, expressionFn, statementsFn func(), out *strings.Builder) {
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

var ifCounter = 0

func ifStatement(token *tokenizer.Token, ifFn, elseFn func(), out *strings.Builder) {
	ifFalse := fmt.Sprintf("IF_%d", ifCounter)
	ifEnd := fmt.Sprintf("IF_END_%d", ifCounter)
	labelFalse := fmt.Sprintf("label %s\n", ifFalse)
	labelEnd := fmt.Sprintf("label %s\n", ifEnd)

	out.WriteString("not\n")
	out.WriteString(fmt.Sprintf("if-goto %s\n", ifFalse))
	ifFn() // compiled statments
	out.WriteString(fmt.Sprintf("goto %s\n", ifEnd))
	out.WriteString(labelFalse)
	elseFn() // compiled statements
	out.WriteString(labelEnd)

	ifCounter++
}
