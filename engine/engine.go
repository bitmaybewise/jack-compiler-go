package engine

import (
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

var (
	notClassVarDec = errors.New("not a class variable declaration")
	notSubroutine  = errors.New("not a subroutine declaration")
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

func CompileClass(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	classToken, err := processToken(tk, is("class"))
	if err != nil {
		return nil, err
	}
	nestedToken := makeNestedToken(classToken)
	nestedToken.append(classToken)

	classNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	nestedToken.append(classNameToken)

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

	for {
		subRoutineToken, err := CompileSubroutine(tk)
		if errors.Is(err, notSubroutine) {
			break
		}
		if err != nil {
			return nil, err
		}
		nestedToken.append(subRoutineToken)
	}

	closeToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	nestedToken.append(closeToken)

	return nestedToken, nil
}

func CompileTerm(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "term"})

	unaryOp := func() (*NestedToken, error) {
		opToken, err := processToken(tk, isUnaryOp())
		if err != nil {
			return nil, err
		}
		nestedToken.append(opToken)

		termToken, err := CompileTerm(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(termToken)

		return nestedToken, nil
	}

	// (expression)
	if _, ok := is("(")(tk.Current); ok {
		openToken, err := processToken(tk, is("("))
		if err != nil {
			return nil, err
		}
		nestedToken.append(openToken)

		if _, ok := isUnaryOp()(tk.Current); ok {
			unaryOp()
		} else {
			expToken, err := CompileExpression(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.append(expToken)
		}

		closeToken, err := processToken(tk, is(")"))
		if err != nil {
			return nil, err
		}
		nestedToken.append(closeToken)

		return nestedToken, nil
	}

	// varName
	termToken, err := processToken(tk, isTerm())
	if err != nil {
		return nil, err
	}
	nestedToken.append(termToken)

	// varName[expression]
	if _, ok := is("[")(tk.Current); ok {
		openArrayToken, err := processToken(tk, is("["))
		if err != nil {
			return nil, err
		}
		nestedToken.append(openArrayToken)

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(expToken)

		closeArrayToken, err := processToken(tk, is("]"))
		if err != nil {
			return nil, err
		}
		nestedToken.append(closeArrayToken)

		return nestedToken, nil
	}

	// unaryOp term
	if _, ok := isUnaryOp()(tk.Current); ok {
		return unaryOp()
	}

	// subroutineCall
	if _, ok := is(".")(tk.Current); ok {
		dotToken, err := processToken(tk, is("."))
		if err != nil {
			return nil, err
		}
		nestedToken.append(dotToken)

		subroutineNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.append(subroutineNameToken)

		openToken, err := processToken(tk, is("("))
		if err != nil {
			return nil, err
		}
		nestedToken.append(openToken)

		expToken, err := CompileExpressionList(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(expToken)

		closeToken, err := processToken(tk, is(")"))
		if err != nil {
			return nil, err
		}
		nestedToken.append(closeToken)

		return nestedToken, nil
	}

	return nestedToken, nil
}

func CompileClassVarDec(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "classVarDec"})

	matcher := or(is("static"), is("field"))
	if _, ok := matcher(tk.Current); !ok {
		return nil, notClassVarDec
	}

	classVarDecToken, err := processToken(tk, matcher)
	if err != nil {
		return nil, err
	}

	nestedToken.append(classVarDecToken)

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}
	nestedToken.append(typeToken)

	for {
		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.append(varNameToken)

		commaToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.append(commaToken)
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
		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.append(varNameToken)

		commaToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.append(commaToken)
	}

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(semicolonToken)

	return nestedToken, nil
}

func CompileSubroutine(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "subroutineDec"})

	matcher := or(is("constructor"), is("function"), is("method"))
	if _, ok := matcher(tk.Current); !ok {
		return nil, notSubroutine
	}

	subRoutineDecToken, err := processToken(tk, matcher)
	if err != nil {
		return nil, err
	}
	nestedToken.append(subRoutineDecToken)

	subRoutineTypeToken, err := processToken(tk, is("void"), isType())
	if err != nil {
		return nil, err
	}
	nestedToken.append(subRoutineTypeToken)

	subRoutineNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	nestedToken.append(subRoutineNameToken)

	openParamToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openParamToken)

	paramsToken, err := CompileParameterList(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(paramsToken)

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

func CompileParameterList(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "parameterList"})

	for {
		if _, ok := isType()(tk.Current); !ok {
			break
		}

		typeToken, err := processToken(tk, isType())
		if err != nil {
			return nil, err
		}
		nestedToken.append(typeToken)

		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.append(varNameToken)

		commaToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.append(commaToken)
	}

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

	statementsToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(statementsToken)

	closeToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeToken)

	return nestedToken, nil
}

func CompileStatements(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "statements"})

	for {
		if _, ok := is("let")(tk.Current); ok {
			token, err := CompileLet(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.append(token)
			continue
		}

		if _, ok := is("if")(tk.Current); ok {
			token, err := CompileIf(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.append(token)
			continue
		}

		if _, ok := is("while")(tk.Current); ok {
			token, err := CompileWhile(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.append(token)
			continue
		}

		if _, ok := is("do")(tk.Current); ok {
			token, err := CompileDo(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.append(token)
			continue
		}

		if _, ok := is("return")(tk.Current); ok {
			returnToken, err := CompileReturn(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.append(returnToken)
			continue
		}

		break
	}

	return nestedToken, nil
}

func CompileWhile(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "whileStatement"})

	returnToken, err := processToken(tk, is("while"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(returnToken)

	openToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	if len(expToken.Children) > 0 {
		nestedToken.append(expToken)
	}

	closeToken, err := processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeToken)

	openStatementToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openStatementToken)

	statementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(statementToken)

	closeStatementToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeStatementToken)

	return nestedToken, nil
}

func CompileReturn(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "returnStatement"})

	returnToken, err := processToken(tk, is("return"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(returnToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	if len(expToken.Children) > 0 {
		nestedToken.append(expToken)
	}

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(semicolonToken)

	return nestedToken, nil
}

func CompileLet(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "letStatement"})

	letToken, err := processToken(tk, is("let"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(letToken)

	termToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	nestedToken.append(termToken)

	if _, ok := is("[")(tk.Current); ok {
		openArrayToken, err := processToken(tk, is("["))
		if err != nil {
			return nil, err
		}
		nestedToken.append(openArrayToken)

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(expToken)

		closeArrayToken, err := processToken(tk, is("]"))
		if err != nil {
			return nil, err
		}
		nestedToken.append(closeArrayToken)
	}

	assignmentToken, err := processToken(tk, is("="))
	if err != nil {
		return nil, err
	}
	nestedToken.append(assignmentToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(expToken)

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(semicolonToken)

	return nestedToken, nil
}

func CompileIf(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "ifStatement"})

	ifToken, err := processToken(tk, is("if"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(ifToken)

	openToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(expToken)

	closeToken, err := processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeToken)

	openStatementToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openStatementToken)

	statementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(statementToken)

	closeStatementToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeStatementToken)

	_, hasElse := is("else")(tk.Current)
	if !hasElse {
		return nestedToken, nil
	}

	elseToken, err := processToken(tk, is("else"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(elseToken)

	openElseStatementToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openElseStatementToken)

	elseStatementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(elseStatementToken)

	closeElseStatementToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(closeElseStatementToken)

	return nestedToken, nil
}

func CompileDo(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "doStatement"})

	doToken, err := processToken(tk, is("do"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(doToken)

	varClassNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	nestedToken.append(varClassNameToken)

	if _, ok := is(".")(tk.Current); ok {
		dotToken, err := processToken(tk, is("."))
		if err != nil {
			return nil, err
		}
		nestedToken.append(dotToken)

		subroutineNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.append(subroutineNameToken)
	}

	openToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.append(openToken)

	expListToken, err := CompileExpressionList(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(expListToken)

	closeToken, err := processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}

	nestedToken.append(closeToken)

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.append(semicolonToken)

	return nestedToken, nil
}

func CompileExpressionList(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "expressionList"})

	if _, ok := is(")")(tk.Current); ok {
		return nestedToken, nil
	}

	for {
		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		if len(expToken.Children) == 0 {
			break
		}
		nestedToken.append(expToken)

		if _, ok := is(",")(tk.Current); !ok {
			break
		}
		commaToken, err := processToken(tk, is(","))
		if err != nil {
			return nil, err
		}
		nestedToken.append(commaToken)
	}

	return nestedToken, nil
}

func CompileExpression(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	nestedToken := makeNestedToken(&tokenizer.Token{Raw: "expression"})

	if _, ok := is(";")(tk.Current); ok {
		return nestedToken, nil
	}

	termNestedToken := makeNestedToken(&tokenizer.Token{Raw: "term"})
	nestedToken.append(termNestedToken)

	termToken, err := CompileTerm(tk)
	if err != nil {
		return nil, err
	}
	termNestedToken.append(termToken)

	for {
		if _, ok := isOp()(tk.Current); !ok {
			break
		}

		opToken, err := processToken(tk, isOp())
		if err != nil {
			return nil, err
		}
		nestedToken.append(opToken)

		termToken, err := CompileTerm(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.append(termToken)
	}

	return nestedToken, nil
}

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
	return or(
		is("+"),
		is("-"),
		is("*"),
		is("/"),
		is("&"),
		is("|"),
		is("<"),
		is(">"),
		is("="),
	)
}

func isUnaryOp() tokenMatcher {
	return or(is("-"), is("~"))
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
			// TODO: subroutineCall

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
