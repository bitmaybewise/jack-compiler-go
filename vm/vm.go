package vm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/logger"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

func Output(compiled *tokenizer.Token, out *strings.Builder) {
	PrintAST(compiled)
	translate(compiled, out)
}

func PrintAST(compiled *tokenizer.Token) {
	logger.Println("printing AST...")

	var line int
	var fn func(*tokenizer.Token, int)
	fn = func(token *tokenizer.Token, tabs int) {
		logger.Printf("%d", line)
		line++

		for i := 0; i < tabs; i++ {
			logger.Print("\t")
		}
		logger.Printf("%+v\n", token)

		for _, child := range token.Children() {
			fn(child, tabs+1)
		}
	}
	fn(compiled, 1)
}

func translate(token *tokenizer.Token, out *strings.Builder) {
	switch {
	case token == nil:
		return

	case token.Kind == "function" || token.Kind == "constructor" || token.Kind == "method":
		function(token, out)

	case token.Kind == "subroutineCall":
		subroutineCall(token, out)
		if token.Parent.Parent.Kind == "do" {
			pop("temp", 0, out)
		}

	case token.Kind == "return":
		for _, expression := range token.Children() {
			translate(expression, out)
		}
		returnCall(token, out)
		return

	case token.Kind == "this":
		this(token, out)
		return

	case token.Kind == "var" && token.Parent.Kind == "let":
		pop("local", token.Var.Index, out)

	case token.Kind == "var" && token.Parent.Kind != "let":
		push("local", token.Var.Index, out)

	case token.Kind == "field" && token.Parent.Kind == "let":
		pop("this", token.Var.Index, out)

	case token.Kind == "field" && token.Parent.Kind != "let":
		push("this", token.Var.Index, out)

	case token.Kind == "static" && token.Parent.Kind == "let":
		pop("static", token.Var.Index, out)

	case token.Kind == "static" && token.Parent.Kind != "let":
		push("static", token.Var.Index, out)

	case token.Kind == "varAssignment":
		n, _ := strconv.Atoi(token.Raw)
		pop("local", n, out)

	case token.Kind == "arg" && token.Parent.Kind == "let":
		pop("argument", token.Var.Index, out)

	case token.Kind == "arg" && token.Parent.Kind != "let":
		push("argument", token.Var.Index, out)

	case token.Kind == "while":
		children := token.Children()
		expression, statements := children[0], children[1]
		while(
			token,
			func() { translate(expression, out) },
			func() { translate(statements, out) },
			out,
		)
		return

	case token.Kind == "if":
		children := token.Children()
		exp := children[0]
		translate(exp, out)
		ifStatement(
			token,
			func() {
				trueStatements := children[1]
				translate(trueStatements, out)
			},
			func() {
				if len(children) >= 3 {
					falseStatements := children[3]
					translate(falseStatements, out)
				}
			},
			out,
		)
		return

	case token.Type == tokenizer.INT_CONST:
		push("constant", token.Raw, out)

	case token.Type == tokenizer.SYMBOL:
		symbol(token, out)

	case token.Type == tokenizer.KEYWORD &&
		token.Raw != "do" &&
		token.Raw != "let":
		keyword(token, out)

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
	logger.Printf("WARNING: ignoring symbol %q\n", token.Raw)
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
	logger.Printf("WARNING: ignoring keyword %q, parent %q\n", token.Raw, token.Parent.Raw)
}

func function(token *tokenizer.Token, out *strings.Builder) {
	cmd := fmt.Sprintf("function %s.%s %d\n", token.Parent.Raw, token.Raw, token.NLocalVars())
	out.WriteString(cmd)

	if token.Kind == "constructor" {
		out.WriteString(fmt.Sprintf("push constant %d\n", token.Parent.NFields))
		out.WriteString("call Memory.alloc 1\n")
		out.WriteString("pop pointer 0\n")
	}
	if token.Kind == "method" {
		out.WriteString("push argument 0\n")
		out.WriteString("pop pointer 0\n")
	}
}

func subroutineCall(token *tokenizer.Token, out *strings.Builder) {
	if token.Method == nil {
		cmd := fmt.Sprintf("call %s %d\n", token.Raw, token.NStackVars())
		out.WriteString(cmd)
		return
	}

	if token.Method.Kind == "field" {
		out.WriteString(fmt.Sprintf("push this %d\n", token.Method.Index))
	}
	// if token.Method.Kind == "class" {
	// 	out.WriteString(fmt.Sprintf("push this %d\n", token.Method.Index))
	// }
	out.WriteString(
		fmt.Sprintf("call %s.%s %d\n", token.Method.Type, token.Raw, token.NStackVars()+1),
	)
}

func returnCall(token *tokenizer.Token, out *strings.Builder) {
	subroutineType := token.Parent.Parent.Parent.Kind
	if subroutineType == "void" {
		push("constant", 0, out)
	}
	out.WriteString("return\n")
}

func this(token *tokenizer.Token, out *strings.Builder) {
	out.WriteString("push pointer 0\n")
}

var whileCounter = 0

func while(token *tokenizer.Token, expressionFn, statementsFn func(), out *strings.Builder) {
	t := fmt.Sprintf("WHILE_EXP_%d", whileCounter)
	f := fmt.Sprintf("WHILE_END_%d", whileCounter)
	labelT := fmt.Sprintf("label %s\n", t)
	labelF := fmt.Sprintf("label %s\n", f)
	whileCounter++

	out.WriteString(labelT)
	expressionFn() // compiled expression
	out.WriteString("not\n")
	out.WriteString(fmt.Sprintf("if-goto %s\n", f))
	statementsFn() // compiled statements
	out.WriteString(fmt.Sprintf("goto %s\n", t))
	out.WriteString(labelF)
}

var ifCounter = 0

func ifStatement(token *tokenizer.Token, ifFn, elseFn func(), out *strings.Builder) {
	ifFalse := fmt.Sprintf("IF_%d", ifCounter)
	ifEnd := fmt.Sprintf("IF_END_%d", ifCounter)
	labelFalse := fmt.Sprintf("label %s\n", ifFalse)
	labelEnd := fmt.Sprintf("label %s\n", ifEnd)
	ifCounter++

	out.WriteString("not\n")
	out.WriteString(fmt.Sprintf("if-goto %s\n", ifFalse))
	ifFn() // compiled statments
	out.WriteString(fmt.Sprintf("goto %s\n", ifEnd))
	out.WriteString(labelFalse)
	elseFn() // compiled statements
	out.WriteString(labelEnd)
}
