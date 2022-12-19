package engine

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hlmerscher/jack-compiler-go/logger"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
	"github.com/hlmerscher/jack-compiler-go/vm"
)

var (
	notClassVarDec   = errors.New("not a class variable declaration")
	notLocalVarDec   = errors.New("not a local variable declaration")
	notSubroutineDec = errors.New("not a subroutine declaration")
	notExpressionDec = errors.New("not an expression declaration")
)

type tokenMatcher func(tokenizer.Token) (string, bool)

func is(tokenTerm string) tokenMatcher {
	return func(t tokenizer.Token) (string, bool) {
		return tokenTerm, t.Raw == tokenTerm
	}
}

func isType() tokenMatcher {
	return or(is("boolean"), is("int"), is("char"), isIdentifier())
}

func isOp() tokenMatcher {
	ops := vm.ArithmeticOps()
	matchers := make([]tokenMatcher, len(ops))
	for i, op := range ops {
		matchers[i] = is(op)
	}
	return or(matchers...)
}

func isUnaryOp() tokenMatcher {
	ops := vm.UnaryOps()
	matchers := make([]tokenMatcher, len(ops))
	for i, op := range ops {
		matchers[i] = is(op)
	}
	return or(matchers...)
}

func isIdentifier() tokenMatcher {
	return func(token tokenizer.Token) (string, bool) {
		matcher := regexp.MustCompile(`^[a-z_A-Z]{1}[a-zA-Z_0-9]*$`)
		itIs := token.Type == tokenizer.IDENTIFIER &&
			matcher.Match([]byte(token.Raw))

		return token.Raw, itIs
	}
}

func isTerm() tokenMatcher {
	return func(token tokenizer.Token) (string, bool) {
		_, isId := isIdentifier()(token)

		itIs := token.Type == tokenizer.INT_CONST ||
			token.Type == tokenizer.STRING_CONST ||
			token.Type == tokenizer.KEYWORD ||
			isId

		return token.Raw, itIs
	}
}

func or(matchers ...tokenMatcher) tokenMatcher {
	return func(t tokenizer.Token) (string, bool) {
		for _, match := range matchers {
			if token, ok := match(t); ok {
				return token, ok
			}
		}
		return t.Raw, false
	}
}

func processToken(tk *tokenizer.Tokenizer, matchers ...tokenMatcher) (*tokenizer.Token, error) {
	var expToken string
	var tokenNames []string

	for _, match := range matchers {
		token, ok := match(tk.Current)
		tokenNames = append(tokenNames, token)
		if ok {
			expToken = token
		}
	}

	if expToken == "" {
		return nil, fmt.Errorf(
			"wrong token error on line %d\n%q\nexpected %q, got %q",
			tk.LineNr, tk.CurrentLine, tokenNames, tk.Current.Raw,
		)
	}

	token := tk.Current

	_, err := tk.Advance()
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func processTokenOrPanics(tk *tokenizer.Tokenizer, matchers ...tokenMatcher) *tokenizer.Token {
	token, err := processToken(tk, matchers...)
	logger.Error(err)
	return token
}
