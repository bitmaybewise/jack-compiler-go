package engine

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

var (
	notClassVarDec = errors.New("not a class variable declaration")
)

type NestedToken struct {
	XMLName  xml.Name
	Children []any
}

func (nt *NestedToken) append(token any) []any {
	if token != nil {
		nt.Children = append(nt.Children, token)
	}
	return nt.Children
}

func makeNestedToken(token *tokenizer.Token) *NestedToken {
	return &NestedToken{XMLName: xml.Name{Local: token.Raw}}
}

var classNames = map[string]struct{}{}

func CompileClass(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	classToken, err := processToken(tk, is("class"))
	if err != nil {
		return nil, err
	}
	nestedToken := makeNestedToken(classToken)
	nestedToken.append(classToken)

	termToken, err := CompileTerm(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(termToken)
	classNames[termToken.Raw] = struct{}{}

	openToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openToken)

	for {
		varDecToken, err := CompileClassVarDec(tk)
		if errors.Is(err, notClassVarDec) {
			break
		}
		if err != nil {
			return nil, err
		}
		nestedToken.append(varDecToken)
	}

	subRoutineToken, err := CompileSubroutine(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(subRoutineToken)

	closeToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeToken)

	return nestedToken, nil
}

func CompileTerm(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	if tk.Current.Type != tokenizer.IDENTIFIER {
		return nil, fmt.Errorf("wrong identifier error, value %s, type <%s>", tk.Current.Raw, tk.Current.Type)
	}
	token := tk.Current

	_, err := tk.Advance()
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func CompileClassVarDec(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	classVarDecToken, err := processToken(tk, is("static"), is("field"))
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", notClassVarDec, err)
	}

	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "classDecVar"})
	nestedToken.append(classVarDecToken)

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}
	nestedToken.append(typeToken)

	for {
		termToken, err := CompileTerm(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(termToken)

		colonToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.append(colonToken)
	}

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(semicolonToken)

	return nestedToken, nil
}

func CompileVarDec(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "varDec"})

	varDecToken, err := processToken(tk, is("var"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(varDecToken)

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}
	nestedToken.append(typeToken)

	for {
		termToken, err := CompileTerm(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(termToken)

		colonToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.append(colonToken)
	}

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(semicolonToken)

	return nestedToken, nil
}

func CompileSubroutine(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	subRoutineDecToken, err := processToken(tk, is("constructor"), is("function"), is("method"))
	if err != nil {
		return nil, err
	}

	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "subroutineDec"})
	nestedToken.append(subRoutineDecToken)

	subRoutineTypeToken, err := processToken(tk, is("void"), isType())
	if err != nil {
		return nil, err
	}
	nestedToken.append(subRoutineTypeToken)

	subRoutineNameToken, err := CompileTerm(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(subRoutineNameToken)

	openParamToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openParamToken)

	// parameter list

	closeParamToken, err := processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeParamToken)

	bodyToken, err := CompileSubroutineBody(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(bodyToken)

	return nestedToken, nil
}

func CompileSubroutineBody(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "subroutineBody"})

	openToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openToken)

	for {
		varToken, err := CompileVarDec(tk)
		if err != nil {
			break
		}
		nestedToken.append(varToken)
	}

	closeToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeToken)

	return nestedToken, nil
}

type tokenMatcher func(tokenizer.Token) (string, bool)

func is(tokenTerm string) tokenMatcher {
	return func(t tokenizer.Token) (string, bool) {
		return tokenTerm, t.Raw == tokenTerm
	}
}

func isClass() tokenMatcher {
	return func(t tokenizer.Token) (string, bool) {
		_, ok := classNames[t.Raw]
		return t.Raw, ok
	}
}

func isType() tokenMatcher {
	return func(t tokenizer.Token) (string, bool) {
		matchers := []tokenMatcher{is("boolean"), is("int"), is("char"), isClass()}
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
		return nil, fmt.Errorf("wrong token error, expected %q, got %q", tokenNames, tk.Current.Raw)
	}

	token := tk.Current

	_, err := tk.Advance()
	if err != nil {
		return nil, err
	}

	return &token, nil
}
