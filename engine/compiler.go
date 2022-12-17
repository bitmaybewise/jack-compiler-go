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
	processTokenOrPanics(tk, is("class"))
	classNameToken := processTokenOrPanics(tk, isIdentifier())
	processTokenOrPanics(tk, is("{"))

	for {
		err := c.Subroutine(tk, classNameToken)
		if errors.Is(err, notSubroutineDec) {
			break
		}
		logger.Error(err)
	}

	processTokenOrPanics(tk, is("}"))
	return nil
}

func (c *Compiler) Subroutine(tk *tokenizer.Tokenizer, classToken *tokenizer.Token) error {
	c.subroutineSymbolTable = make(map[string]*tokenizer.Var)

	matcher := or(is("constructor"), is("function"), is("method"))
	if _, ok := matcher(tk.Current); !ok {
		return notSubroutineDec
	}
	processTokenOrPanics(tk, matcher)

	typeToken := processTokenOrPanics(tk, is("void"), isType())
	nameToken := processTokenOrPanics(tk, isIdentifier())

	processTokenOrPanics(tk, is("("))
	var args int
	c.ParameterList(tk, &args)
	processTokenOrPanics(tk, is(")"))

	err := c.SubroutineBody(tk, classToken, nameToken, typeToken)
	if err != nil {
		return err
	}

	return nil
}

func (c *Compiler) SubroutineBody(tk *tokenizer.Tokenizer, className, subroutineName, subroutineType *tokenizer.Token) error {
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

	c.vmw.WriteSubroutine(className.Raw, subroutineName.Raw, nvars)

	c.Statements(tk, nvars, subroutineType)
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

func (c *Compiler) Statements(tk *tokenizer.Tokenizer, nvars int, subroutineType *tokenizer.Token) error {
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
			if err := c.Do(tk, nvars); err != nil {
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
			if err := c.Statements(tk, 0, nil); err != nil {
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
			if err := c.Statements(tk, 0, nil); err != nil {
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
			if err := c.Statements(tk, 0, nil); err != nil {
				return err
			}
			processTokenOrPanics(tk, is("}"))

			return nil
		},
	)

	return nil
}

func (c *Compiler) Do(tk *tokenizer.Tokenizer, nvars int) error {
	var varClassNameToken, subroutineNameToken *tokenizer.Token

	processTokenOrPanics(tk, is("do"))
	varClassNameToken = processTokenOrPanics(tk, isIdentifier())

	if _, ok := is(".")(tk.Current); ok {
		processTokenOrPanics(tk, is("."))
		subroutineNameToken = processTokenOrPanics(tk, isIdentifier())
	}

	processTokenOrPanics(tk, is("("))
	n, err := c.ExpressionList(tk)
	if err != nil {
		return err
	}
	processTokenOrPanics(tk, is(")"))
	processTokenOrPanics(tk, is(";"))
	c.vmw.WriteCall(varClassNameToken.Raw, subroutineNameToken.Raw, n)
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

	processTokenOrPanics(tk, is("="))
	err = c.Expression(tk)
	if err != nil {
		return err
	}
	c.vmw.WritePop(vm.VarTypes[_var.Kind], _var.Index)
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
	if _, ok := is(";")(tk.Current); ok {
		return notExpressionDec
	}

	err := c.Term(tk)
	if err != nil {
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

	// varName
	termToken := processTokenOrPanics(tk, isTerm())
	_var, err := c.enforceVarDec(tk, termToken)
	if err != nil {
		return err
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
		c.vmw.WriteCall(termToken.Raw, subroutineNameToken.Raw, n)

		return nil
	}

	if termToken.Type == tokenizer.IDENTIFIER {
		c.vmw.WritePush(vm.VarTypes[_var.Kind], _var.Index)
	}
	if termToken.Type == tokenizer.INT_CONST {
		c.vmw.WritePush("constant", termToken.Raw)
	}
	if termToken.Type == tokenizer.KEYWORD {
		c.vmw.WriteKeyword(termToken.Raw)
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
		vmw:                   buf,
		subroutineSymbolTable: make(map[string]*tokenizer.Var),
	}
}
