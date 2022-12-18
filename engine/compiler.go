package engine

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hlmerscher/jack-compiler-go/logger"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
	"github.com/hlmerscher/jack-compiler-go/vm"
)

type Compiler struct {
	vmw *vm.Writer

	classSymbolTable      map[string]*tokenizer.Var
	subroutineSymbolTable map[string]*tokenizer.Var
}

func (c *Compiler) Class(tk *tokenizer.Tokenizer) error {
	c.classSymbolTable = make(map[string]*tokenizer.Var)

	processTokenOrPanics(tk, is("class"))
	classNameToken := processTokenOrPanics(tk, isIdentifier())
	c.classSymbolTable["this"] = &tokenizer.Var{
		Index: 0,
		Type:  classNameToken.Raw,
		Kind:  "class",
	}

	processTokenOrPanics(tk, is("{"))
	var nvars int
	for {
		err := c.ClassVarDec(tk, &nvars)
		if errors.Is(err, notClassVarDec) {
			break
		}
		if err != nil {
			return err
		}
	}
	for {
		err := c.Subroutine(tk, classNameToken, nvars)
		if errors.Is(err, notSubroutineDec) {
			break
		}
		logger.Error(err)
	}
	processTokenOrPanics(tk, is("}"))

	return nil
}

func (c *Compiler) ClassVarDec(tk *tokenizer.Tokenizer, nvars *int) error {
	matcher := or(is("static"), is("field"))
	if _, ok := matcher(tk.Current); !ok {
		return notClassVarDec
	}

	classVarDecToken := processTokenOrPanics(tk, matcher)
	typeToken := processTokenOrPanics(tk, isType())

	for i := 0; ; i++ {
		varNameToken := processTokenOrPanics(tk, isIdentifier())
		c.classSymbolTable[varNameToken.Raw] = &tokenizer.Var{
			Type:  typeToken.Raw,
			Kind:  classVarDecToken.Raw,
			Index: *nvars,
		}
		*nvars++

		if _, err := processToken(tk, is(",")); err != nil {
			break
		}
	}
	processTokenOrPanics(tk, is(";"))

	return nil
}

func (c *Compiler) Subroutine(tk *tokenizer.Tokenizer, classToken *tokenizer.Token, nClassVars int) error {
	c.subroutineSymbolTable = make(map[string]*tokenizer.Var)

	matcher := or(is("constructor"), is("function"), is("method"))
	if _, ok := matcher(tk.Current); !ok {
		return notSubroutineDec
	}
	_, isConstructor := is("constructor")(tk.Current)
	_, isMethod := is("method")(tk.Current)
	processTokenOrPanics(tk, matcher)

	typeToken := processTokenOrPanics(tk, is("void"), isType())
	nameToken := processTokenOrPanics(tk, isIdentifier())

	processTokenOrPanics(tk, is("("))
	var args int
	if isMethod {
		args++
	}
	c.ParameterList(tk, &args)
	processTokenOrPanics(tk, is(")"))

	writeSubroutine := func(nLocalVars int) {
		c.vmw.WriteSubroutine(classToken.Raw, nameToken.Raw, nLocalVars)
		if isConstructor {
			c.vmw.WritePush("constant", nClassVars)
			c.vmw.WriteCall("Memory", "alloc", 1)
			c.vmw.WritePop("pointer", 0)
		}
		if isMethod {
			c.vmw.WritePush("argument", 0)
			c.vmw.WritePop("pointer", 0)
		}
	}

	err := c.SubroutineBody(tk, typeToken, writeSubroutine)
	if err != nil {
		return err
	}

	return nil
}

func (c *Compiler) SubroutineBody(tk *tokenizer.Tokenizer, subroutineType *tokenizer.Token, writeSubroutine func(int)) error {
	processTokenOrPanics(tk, is("{"))

	var nvars int
	for varLine := 0; ; varLine++ {
		err := c.VarDec(tk, &nvars)
		if errors.Is(err, notLocalVarDec) {
			break
		}
		if err != nil {
			return err
		}
	}
	writeSubroutine(nvars)

	c.Statements(tk, subroutineType)
	processTokenOrPanics(tk, is("}"))

	return nil
}

func (c *Compiler) VarDec(tk *tokenizer.Tokenizer, nvars *int) error {
	_, err := processToken(tk, is("var"))
	if err != nil {
		return fmt.Errorf("%w: %s", notLocalVarDec, err)
	}
	typeToken := processTokenOrPanics(tk, isType())

	for {
		varNameToken := processTokenOrPanics(tk, isIdentifier())
		c.subroutineSymbolTable[varNameToken.Raw] = &tokenizer.Var{
			Index: *nvars,
			Type:  typeToken.Raw,
			Kind:  "var",
		}
		*nvars++

		_, err = processToken(tk, is(","))
		if err != nil {
			break
		}
	}
	processTokenOrPanics(tk, is(";"))

	return nil
}

func (c *Compiler) Statements(tk *tokenizer.Tokenizer, subroutineType *tokenizer.Token) error {
	for {
		if _, ok := is("let")(tk.Current); ok {
			if err := c.Let(tk); err != nil {
				return err
			}
			continue
		}
		if _, ok := is("if")(tk.Current); ok {
			if err := c.If(tk); err != nil {
				return err
			}
			continue
		}
		if _, ok := is("while")(tk.Current); ok {
			if err := c.While(tk); err != nil {
				return err
			}
			continue
		}
		if _, ok := is("do")(tk.Current); ok {
			if err := c.Do(tk); err != nil {
				return err
			}
			continue
		}
		if _, ok := is("return")(tk.Current); ok {
			if err := c.Return(tk, subroutineType); err != nil {
				return err
			}
			continue
		}

		break
	}

	return nil
}

func (c *Compiler) While(tk *tokenizer.Tokenizer) error {
	processTokenOrPanics(tk, is("while"))
	c.vmw.WriteWhile(
		func() error {
			processTokenOrPanics(tk, is("("))
			if err := c.Expression(tk); err != nil {
				return err
			}
			processTokenOrPanics(tk, is(")"))

			return nil
		},
		func() error {
			processTokenOrPanics(tk, is("{"))
			if err := c.Statements(tk, nil); err != nil {
				return err
			}
			processTokenOrPanics(tk, is("}"))

			return nil
		},
	)

	return nil
}

func (c *Compiler) If(tk *tokenizer.Tokenizer) error {
	processTokenOrPanics(tk, is("if"))
	processTokenOrPanics(tk, is("("))
	if err := c.Expression(tk); err != nil {
		return err
	}
	processTokenOrPanics(tk, is(")"))
	processTokenOrPanics(tk, is("{"))

	c.vmw.WriteIf(
		func() error {
			if err := c.Statements(tk, nil); err != nil {
				return err
			}
			processTokenOrPanics(tk, is("}"))

			return nil
		},
		func() error {
			_, hasElse := is("else")(tk.Current)
			if !hasElse {
				return nil
			}
			processTokenOrPanics(tk, is("else"))
			processTokenOrPanics(tk, is("{"))
			if err := c.Statements(tk, nil); err != nil {
				return err
			}
			processTokenOrPanics(tk, is("}"))

			return nil
		},
	)

	return nil
}

func (c *Compiler) Do(tk *tokenizer.Tokenizer) error {
	processTokenOrPanics(tk, is("do"))
	if err := c.Expression(tk); err != nil {
		return err
	}
	processTokenOrPanics(tk, is(";"))

	c.vmw.WritePop("temp", 0)

	return nil
}

func (c *Compiler) Let(tk *tokenizer.Tokenizer) error {
	processTokenOrPanics(tk, is("let"))

	termToken := processTokenOrPanics(tk, isIdentifier())
	_var, err := c.enforceVarDec(tk, termToken)
	if err != nil {
		return err
	}

	if _, ok := is("[")(tk.Current); ok {
		c.vmw.WritePush(vm.VarTypes[_var.Kind], _var.Index)

		processTokenOrPanics(tk, is("["))
		if err := c.Expression(tk); err != nil {
			return err
		}
		processTokenOrPanics(tk, is("]"))

		c.vmw.WriteArithmetic("+")

		processTokenOrPanics(tk, is("="))
		if err = c.Expression(tk); err != nil {
			return err
		}

		c.vmw.WritePop("temp", 0)
		c.vmw.WritePop("pointer", 1)
		c.vmw.WritePush("temp", 0)
		c.vmw.WritePop("that", 0)

	} else {
		processTokenOrPanics(tk, is("="))
		if err = c.Expression(tk); err != nil {
			return err
		}
		c.vmw.WritePop(vm.VarTypes[_var.Kind], _var.Index)
	}

	processTokenOrPanics(tk, is(";"))

	return nil
}

func (c *Compiler) ExpressionList(tk *tokenizer.Tokenizer) (int, error) {
	var n int

	if _, ok := is(")")(tk.Current); ok {
		return n, nil
	}

	for {
		err := c.Expression(tk)
		if errors.Is(err, notExpressionDec) {
			break
		}
		if err != nil {
			return n, err
		}
		n++

		if _, ok := is(",")(tk.Current); !ok {
			break
		}
		processTokenOrPanics(tk, is(","))
	}

	return n, nil
}

func (c *Compiler) Expression(tk *tokenizer.Tokenizer) error {
	if _, ok := or(is(";"), is(")"))(tk.Current); ok {
		return notExpressionDec
	}

	if err := c.Term(tk); err != nil {
		return err
	}

	for {
		if _, ok := isOp()(tk.Current); !ok {
			break
		}

		opToken := processTokenOrPanics(tk, isOp())

		err := c.Term(tk)
		if err != nil {
			return err
		}

		c.vmw.WriteArithmetic(opToken.Raw)
	}

	return nil
}

func (c *Compiler) Term(tk *tokenizer.Tokenizer) error {
	// unaryOp term
	if _, ok := isUnaryOp()(tk.Current); ok {
		opToken := processTokenOrPanics(tk, isUnaryOp())
		if err := c.Term(tk); err != nil {
			return err
		}
		c.vmw.WriteUnary(opToken.Raw)

		return nil
	}

	// (expression)
	if _, ok := is("(")(tk.Current); ok {
		processTokenOrPanics(tk, is("("))
		if err := c.Expression(tk); err != nil {
			return err
		}
		processTokenOrPanics(tk, is(")"))

		return nil
	}

	// varName / methodName
	termToken := processTokenOrPanics(tk, isTerm())

	// (method call)
	if _, ok := is("(")(tk.Current); ok {
		processTokenOrPanics(tk, is("("))
		n, err := c.ExpressionList(tk)
		if err != nil && !errors.Is(err, notExpressionDec) {
			return err
		}
		processTokenOrPanics(tk, is(")"))

		_var, ok := c.classSymbolTable["this"]
		if !ok {
			logger.Errorf("not a method call: %q\n", termToken.Raw)
		}
		c.vmw.WritePush("pointer", 0)
		c.vmw.WriteCall(_var.Type, termToken.Raw, n+1) // +1, given this is pushed to the stack

		return nil
	}

	_var, err := c.enforceVarDec(tk, termToken)
	if err != nil {
		return err
	}

	// [expression]
	if _, ok := is("[")(tk.Current); ok {
		c.vmw.WritePush(vm.VarTypes[_var.Kind], _var.Index)

		processTokenOrPanics(tk, is("["))
		if err := c.Expression(tk); err != nil {
			return err
		}
		processTokenOrPanics(tk, is("]"))

		c.vmw.WriteArithmetic("+")
		c.vmw.WritePop("pointer", 1)
		c.vmw.WritePush("that", 0)

		return nil
	}

	// subroutineCall
	if _, ok := is(".")(tk.Current); ok {
		processTokenOrPanics(tk, is("."))
		subroutineNameToken := processTokenOrPanics(tk, isIdentifier())
		processTokenOrPanics(tk, is("("))
		n, err := c.ExpressionList(tk)
		if err != nil {
			return err
		}
		processTokenOrPanics(tk, is(")"))

		if _var != nil {
			c.vmw.WritePush(vm.VarTypes[_var.Kind], _var.Index)
			c.vmw.WriteCall(_var.Type, subroutineNameToken.Raw, n+1) // method call, previous push instruction is pushing obj to the stack
		} else {
			c.vmw.WriteCall(termToken.Raw, subroutineNameToken.Raw, n)
		}

		return nil
	}

	if termToken.Type == tokenizer.IDENTIFIER {
		c.vmw.WritePush(vm.VarTypes[_var.Kind], _var.Index)
	}
	if termToken.Type == tokenizer.INT_CONST {
		c.vmw.WritePush("constant", termToken.Raw)
	}
	if termToken.Type == tokenizer.STRING_CONST {
		c.vmw.WritePush("constant", len(termToken.Raw)-2)
		c.vmw.WriteCall("String", "new", 1)
		for _, char := range termToken.Raw {
			if char == '"' {
				continue
			}
			c.vmw.WritePush("constant", char)
			c.vmw.WriteCall("String", "appendChar", 2) // 2 because 1 is the string ref, 1 is the char
		}
	}
	if termToken.Type != tokenizer.KEYWORD {
		return nil
	}
	if _var == nil {
		c.vmw.WriteKeyword(termToken.Raw)
	} else {
		c.vmw.WritePush(vm.VarTypes[_var.Kind], _var.Index)
	}

	return nil
}

func (c *Compiler) Return(tk *tokenizer.Tokenizer, subroutineType *tokenizer.Token) error {
	processTokenOrPanics(tk, is("return"))
	err := c.Expression(tk)
	if err != nil && !errors.Is(err, notExpressionDec) {
		return err
	}
	processTokenOrPanics(tk, is(";"))

	if subroutineType != nil && subroutineType.Raw == "void" {
		c.vmw.WritePush("constant", 0)
	}
	c.vmw.WriteReturn()

	return nil
}

func (c *Compiler) ParameterList(tk *tokenizer.Tokenizer, args *int) error {
	for i := *args; ; i++ {
		if _, ok := isType()(tk.Current); !ok {
			break
		}

		typeToken := processTokenOrPanics(tk, isType())
		varNameToken := processTokenOrPanics(tk, isIdentifier())
		c.subroutineSymbolTable[varNameToken.Raw] = &tokenizer.Var{
			Type:  typeToken.Raw,
			Kind:  "arg",
			Index: i,
		}

		_, err := processToken(tk, is(","))
		if err != nil {
			break
		}
	}

	return nil
}

func (c *Compiler) enforceVarDec(tk *tokenizer.Tokenizer, termToken *tokenizer.Token) (*tokenizer.Var, error) {
	subroutineSymbol, inSubroutineDec := c.subroutineSymbolTable[termToken.Raw]
	classSymbol, inClassDec := c.classSymbolTable[termToken.Raw]
	found := inSubroutineDec || inClassDec
	// the jack compiler performs no linking, so if the term starts with a uppercased letter,
	// it assumes this class will be available at runtime
	isClassName := regexp.MustCompile("[A-Z].*").Match([]byte(termToken.Raw))

	err := fmt.Errorf("line %d: %q\nvariable %q not declared", tk.LineNr, tk.CurrentLine, termToken.Raw)
	if termToken.Type == tokenizer.IDENTIFIER && !found && !isClassName {
		return nil, err
	}

	if inSubroutineDec {
		return subroutineSymbol, nil
	}
	if inClassDec {
		return classSymbol, nil
	}

	return nil, nil
}

func New(buf *vm.Writer) Compiler {
	return Compiler{
		vmw: buf,
	}
}
