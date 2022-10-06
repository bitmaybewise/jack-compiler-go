package engine

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

var (
	notClassVarDec = errors.New("not a class variable declaration")
	notSubroutine  = errors.New("not a subroutine declaration")
)

type symbol struct {
	_type string
	_kind string
	index int
}

var (
	classSymbolTable      = map[string]symbol{}
	subroutineSymbolTable = map[string]symbol{}
)

type Subroutine struct {
	tokenizer.NestedToken
}

func CompileClass(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	classToken, err := processToken(tk, is("class"))
	if err != nil {
		return nil, err
	}

	classNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	class := tokenizer.MakeNestedToken(classNameToken)
	class.Kind = classToken.Raw

	_, err = processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}

	classSymbolTable = make(map[string]symbol)
	for {
		varDecToken, err := CompileClassVarDec(tk)
		if errors.Is(err, notClassVarDec) {
			break
		}
		if err != nil {
			return nil, err
		}
		class.Append(varDecToken)
	}

	for {
		subRoutineToken, err := CompileSubroutine(tk)
		if errors.Is(err, notSubroutine) {
			break
		}
		if err != nil {
			return nil, err
		}

		class.Append(subRoutineToken)
		subRoutineToken.Parent = class
	}

	_, err = processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	return class, nil
}

func CompileTerm(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "term"})

	// unaryOp term
	if _, ok := isUnaryOp()(tk.Current); ok {
		opToken, err := processToken(tk, isUnaryOp())
		if err != nil {
			return nil, err
		}

		termToken, err := CompileTerm(tk)
		if err != nil {
			return nil, err
		}

		nestedToken.Append(termToken)
		nestedToken.Append(opToken)

		return nestedToken, nil
	}

	// (expression)
	if _, ok := is("(")(tk.Current); ok {
		_, err := processToken(tk, is("("))
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(openToken)

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		_, err = processToken(tk, is(")"))
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(closeToken)

		return nestedToken, nil
	}

	// varName
	termToken, err := processToken(tk, isTerm())
	if err != nil {
		return nil, err
	}
	if err = enforceVarDec(tk, termToken); err != nil {
		return nil, err
	}
	nestedToken.Append(termToken)

	// varName[expression]
	if _, ok := is("[")(tk.Current); ok {
		openArrayToken, err := processToken(tk, is("["))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(openArrayToken)

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		closeArrayToken, err := processToken(tk, is("]"))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(closeArrayToken)

		return nestedToken, nil
	}

	// subroutineCall
	if _, ok := is(".")(tk.Current); ok {
		dotToken, err := processToken(tk, is("."))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(dotToken)

		subroutineNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.Append(subroutineNameToken)

		openToken, err := processToken(tk, is("("))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(openToken)

		expToken, err := CompileExpressionList(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		closeToken, err := processToken(tk, is(")"))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(closeToken)

		return nestedToken, nil
	}

	return nestedToken, nil
}

func CompileClassVarDec(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "classVarDec"})

	matcher := or(is("static"), is("field"))
	if _, ok := matcher(tk.Current); !ok {
		return nil, notClassVarDec
	}

	classVarDecToken, err := processToken(tk, matcher)
	if err != nil {
		return nil, err
	}

	nestedToken.Append(classVarDecToken)

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}
	nestedToken.Append(typeToken)

	for i := 0; ; i++ {
		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.Append(varNameToken)
		classSymbolTable[varNameToken.Raw] = symbol{
			_type: typeToken.Raw,
			_kind: classVarDecToken.Raw,
			index: i,
		}

		commaToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.Append(commaToken)
	}

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(semicolonToken)

	return nestedToken, nil
}

func CompileVarDec(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "varDec"})

	varDecToken, err := processToken(tk, is("var"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(varDecToken)

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}
	nestedToken.Append(typeToken)

	for i := 0; ; i++ {
		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.Append(varNameToken)
		subroutineSymbolTable[varNameToken.Raw] = symbol{
			_type: typeToken.Raw,
			_kind: "var",
			index: i,
		}

		commaToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.Append(commaToken)
	}

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(semicolonToken)

	return nestedToken, nil
}

func CompileSubroutine(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	subroutineSymbolTable = make(map[string]symbol)

	matcher := or(is("constructor"), is("function"), is("method"))
	if _, ok := matcher(tk.Current); !ok {
		return nil, notSubroutine
	}

	subRoutineDecToken, err := processToken(tk, matcher)
	if err != nil {
		return nil, err
	}

	subRoutineTypeToken, err := processToken(tk, is("void"), isType())
	if err != nil {
		return nil, err
	}

	subRoutineNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	nestedToken := tokenizer.MakeNestedToken(subRoutineNameToken)
	nestedToken.Type = subRoutineTypeToken.Raw
	nestedToken.Kind = subRoutineDecToken.Raw

	_, err = processToken(tk, is("("))
	if err != nil {
		return nil, err
	}

	paramsToken, err := CompileParameterList(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(paramsToken)

	_, err = processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}

	bodyToken, err := CompileSubroutineBody(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(bodyToken)
	bodyToken.Parent = nestedToken

	return nestedToken, nil
}

func CompileParameterList(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "parameterList"})

	for i := 0; ; i++ {
		if _, ok := isType()(tk.Current); !ok {
			break
		}

		typeToken, err := processToken(tk, isType())
		if err != nil {
			return nil, err
		}
		nestedToken.Append(typeToken)

		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		nestedToken.Append(varNameToken)
		subroutineSymbolTable[varNameToken.Raw] = symbol{
			_type: typeToken.Raw,
			_kind: "arg",
			index: i,
		}

		commaToken, err := processToken(tk, is(","))
		if err != nil {
			break
		}
		nestedToken.Append(commaToken)
	}

	return nestedToken, nil
}

func CompileSubroutineBody(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "subroutineBody"})

	_, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}

	for {
		varToken, err := CompileVarDec(tk)
		if err != nil {
			break
		}
		nestedToken.Append(varToken)
	}

	statementsToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(statementsToken)
	statementsToken.Parent = nestedToken

	_, err = processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileStatements(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "statements"})

	for {
		if _, ok := is("let")(tk.Current); ok {
			token, err := CompileLet(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.Append(token)
			continue
		}

		if _, ok := is("if")(tk.Current); ok {
			token, err := CompileIf(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.Append(token)
			continue
		}

		if _, ok := is("while")(tk.Current); ok {
			token, err := CompileWhile(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.Append(token)
			continue
		}

		if _, ok := is("do")(tk.Current); ok {
			token, err := CompileDo(tk)
			if err != nil {
				return nil, err
			}
			nestedToken.Append(token)
			continue
		}

		if _, ok := is("return")(tk.Current); ok {
			returnToken, err := CompileReturn(tk)
			if err != nil {
				return nil, err
			}
			returnToken.Parent = nestedToken
			nestedToken.Append(returnToken)
			continue
		}

		break
	}

	return nestedToken, nil
}

func CompileWhile(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "whileStatement"})

	returnToken, err := processToken(tk, is("while"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(returnToken)

	openToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(openToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	if len(expToken.Children()) > 0 {
		nestedToken.Append(expToken)
	}

	closeToken, err := processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(closeToken)

	openStatementToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(openStatementToken)

	statementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(statementToken)

	closeStatementToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(closeStatementToken)

	return nestedToken, nil
}

func CompileReturn(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	returnToken, err := processToken(tk, is("return"))
	if err != nil {
		return nil, err
	}
	nestedToken := tokenizer.MakeNestedToken(returnToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	if len(expToken.Children()) > 0 {
		nestedToken.Append(expToken)
	}

	_, err = processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileLet(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "letStatement"})

	letToken, err := processToken(tk, is("let"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(letToken)

	termToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	if err = enforceVarDec(tk, termToken); err != nil {
		return nil, err
	}
	nestedToken.Append(termToken)

	if _, ok := is("[")(tk.Current); ok {
		openArrayToken, err := processToken(tk, is("["))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(openArrayToken)

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		closeArrayToken, err := processToken(tk, is("]"))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(closeArrayToken)
	}

	assignmentToken, err := processToken(tk, is("="))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(assignmentToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(expToken)

	semicolonToken, err := processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(semicolonToken)

	return nestedToken, nil
}

func enforceVarDec(tk *tokenizer.Tokenizer, termToken *tokenizer.Token) error {
	_, inSubroutineDec := subroutineSymbolTable[termToken.Raw]
	_, inClassDec := classSymbolTable[termToken.Raw]
	found := inSubroutineDec || inClassDec

	if termToken.Type == tokenizer.IDENTIFIER && !found {
		return fmt.Errorf("line %d: %q\nvariable %q not declared", tk.LineNr, tk.CurrentLine, termToken.Raw)
	}

	return nil
}

func CompileIf(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "ifStatement"})

	ifToken, err := processToken(tk, is("if"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(ifToken)

	openToken, err := processToken(tk, is("("))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(openToken)

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(expToken)

	closeToken, err := processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(closeToken)

	openStatementToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(openStatementToken)

	statementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(statementToken)

	closeStatementToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(closeStatementToken)

	_, hasElse := is("else")(tk.Current)
	if !hasElse {
		return nestedToken, nil
	}

	elseToken, err := processToken(tk, is("else"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(elseToken)

	openElseStatementToken, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(openElseStatementToken)

	elseStatementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(elseStatementToken)

	closeElseStatementToken, err := processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(closeElseStatementToken)

	return nestedToken, nil
}

func CompileDo(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	doToken, err := processToken(tk, is("do"))
	if err != nil {
		return nil, err
	}
	do := tokenizer.MakeNestedToken(doToken)

	varClassNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	do.Append(varClassNameToken)

	if _, ok := is(".")(tk.Current); ok {
		dotToken, err := processToken(tk, is("."))
		if err != nil {
			return nil, err
		}
		do.Append(dotToken)

		subroutineNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		do.Append(subroutineNameToken)
	}

	_, err = processToken(tk, is("("))
	if err != nil {
		return nil, err
	}

	expListToken, err := CompileExpressionList(tk)
	if err != nil {
		return nil, err
	}
	do.Append(expListToken)

	_, err = processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}

	_, err = processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}

	return do, nil
}

func CompileExpressionList(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "expressionList"})

	if _, ok := is(")")(tk.Current); ok {
		return nestedToken, nil
	}

	for {
		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		if len(expToken.Children()) == 0 {
			break
		}
		nestedToken.Append(expToken)

		if _, ok := is(",")(tk.Current); !ok {
			break
		}
		commaToken, err := processToken(tk, is(","))
		if err != nil {
			return nil, err
		}
		nestedToken.Append(commaToken)
	}

	return nestedToken, nil
}

func CompileExpression(tk *tokenizer.Tokenizer) (*tokenizer.NestedToken, error) {
	nestedToken := tokenizer.MakeNestedToken(&tokenizer.Token{Raw: "expression"})

	if _, ok := is(";")(tk.Current); ok {
		return nestedToken, nil
	}

	termToken, err := CompileTerm(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(termToken)

	for {
		if _, ok := isOp()(tk.Current); !ok {
			break
		}

		opToken, err := processToken(tk, isOp())
		if err != nil {
			return nil, err
		}

		termToken, err := CompileTerm(tk)
		if err != nil {
			return nil, err
		}

		nestedToken.Append(termToken)
		nestedToken.Append(opToken)
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
