package engine

import (
	"errors"

	"github.com/hlmerscher/jack-compiler-go/logger"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
	"github.com/hlmerscher/jack-compiler-go/vm"
)

type Compiler struct {
	vmw *vm.Writer
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
	matcher := or(is("constructor"), is("function"), is("method"))
	if _, ok := matcher(tk.Current); !ok {
		return notSubroutineDec
	}
	processTokenOrPanics(tk, matcher)

	processTokenOrPanics(tk, is("void"), isType())
	fnNameToken := processTokenOrPanics(tk, isIdentifier())

	processTokenOrPanics(tk, is("("))
	var args int
	c.ParameterList(tk, &args)
	processTokenOrPanics(tk, is(")"))

	c.vmw.WriteSubroutine(classToken.Raw, fnNameToken.Raw, args)

	err := c.SubroutineBody(tk)
	if err != nil {
		return err
	}

	return nil
}

func (c *Compiler) SubroutineBody(tk *tokenizer.Tokenizer) error {
	processTokenOrPanics(tk, is("{"))
	c.Statements(tk)
	processTokenOrPanics(tk, is("}"))

	return nil
}

func (c *Compiler) Statements(tk *tokenizer.Tokenizer) error {
	for {
		if _, ok := is("do")(tk.Current); ok {
			err := c.Do(tk)
			if err != nil {
				return err
			}
			continue
		}

		if _, ok := is("return")(tk.Current); ok {
			err := c.Return(tk)
			if err != nil {
				return err
			}
			continue
		}

		break
	}

	return nil
}

func (c *Compiler) Do(tk *tokenizer.Tokenizer) error {
	var varClassNameToken, subroutineNameToken *tokenizer.Token

	processTokenOrPanics(tk, is("do"))
	varClassNameToken = processTokenOrPanics(tk, isIdentifier())

	if _, ok := is(".")(tk.Current); ok {
		processTokenOrPanics(tk, is("."))
		subroutineNameToken = processTokenOrPanics(tk, isIdentifier())
	}

	processTokenOrPanics(tk, is("("))
	err := c.ExpressionList(tk)
	if err != nil {
		return err
	}
	processTokenOrPanics(tk, is(")"))
	processTokenOrPanics(tk, is(";"))
	c.vmw.WriteCall(varClassNameToken.Raw, subroutineNameToken.Raw, 0)
	c.vmw.WritePop("temp", 0)

	return nil
}

func (c *Compiler) ExpressionList(tk *tokenizer.Tokenizer) error {
	if _, ok := is(")")(tk.Current); ok {
		return nil
	}

	for {
		err := c.Expression(tk)
		if errors.Is(err, notExpressionDec) {
			break
		}
		if err != nil {
			return err
		}

		if _, ok := is(",")(tk.Current); !ok {
			break
		}

		processTokenOrPanics(tk, is(","))
	}

	return nil
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
	// (expression)
	if _, ok := is("(")(tk.Current); ok {
		processTokenOrPanics(tk, is("("))
		err := c.Expression(tk)
		if err != nil {
			return err
		}
		processTokenOrPanics(tk, is(")"))

		return nil
	}

	// varName
	termToken := processTokenOrPanics(tk, isTerm())
	_, err := enforceVarDec(tk, termToken)
	if err != nil {
		return err
	}

	if termToken.Type == tokenizer.INT_CONST {
		c.vmw.WritePush("constant", termToken.Raw)
	}

	return nil
}

func (c *Compiler) Return(tk *tokenizer.Tokenizer) error {
	processTokenOrPanics(tk, is("return"))
	processTokenOrPanics(tk, is(";"))
	c.vmw.WriteReturn()

	return nil
}

func (c *Compiler) ParameterList(tk *tokenizer.Tokenizer, args *int) error {
	return nil
}

func New(buf *vm.Writer) Compiler {
	return Compiler{buf}
}
