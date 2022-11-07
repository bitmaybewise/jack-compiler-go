package engine

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hlmerscher/jack-compiler-go/onerror"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

var (
	notClassVarDec   = errors.New("not a class variable declaration")
	notSubroutineDec = errors.New("not a subroutine declaration")
	notExpressionDec = errors.New("not an expression declaration")
)

var (
	classSymbolTable      map[string]*tokenizer.Var
	subroutineSymbolTable map[string]*tokenizer.Var
)

func makeClassSymbolTable() map[string]*tokenizer.Var {
	return make(map[string]*tokenizer.Var)
}

func CompileClass(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	classSymbolTable = makeClassSymbolTable()

	classToken := processTokenOrPanics(tk, is("class"))

	classNameToken := processTokenOrPanics(tk, isIdentifier())
	class := classNameToken
	class.Kind = classToken.Raw
	classSymbolTable[classNameToken.Raw] = &tokenizer.Var{}

	processTokenOrPanics(tk, is("{"))

	for {
		_, err := CompileClassVarDec(tk)
		if errors.Is(err, notClassVarDec) {
			break
		}
		onerror.Log(err)
	}

	for {
		subRoutineToken, err := CompileSubroutine(tk)
		if errors.Is(err, notSubroutineDec) {
			break
		}
		onerror.Log(err)

		class.Append(subRoutineToken)
	}

	processTokenOrPanics(tk, is("}"))

	return class, nil
}

func CompileTerm(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "term"}

	// unaryOp term
	if _, ok := isUnaryOp()(tk.Current); ok {
		opToken, err := processToken(tk, isUnaryOp())
		if err != nil {
			return nil, err
		}
		opToken.Kind = "unary"

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

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		_, err = processToken(tk, is(")"))
		if err != nil {
			return nil, err
		}

		return nestedToken, nil
	}

	// varName
	termToken, err := processToken(tk, isTerm())
	if err != nil {
		return nil, err
	}
	if _, err = enforceVarDec(tk, termToken); err != nil {
		return nil, err
	}
	// nestedToken.Append(termToken)

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
		// nestedToken.Append(dotToken)
		termToken.Raw += dotToken.Raw

		subroutineNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(subroutineNameToken)
		termToken.Raw += subroutineNameToken.Raw

		_, err = processToken(tk, is("("))
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(openToken)

		expToken, err := CompileExpressionList(tk)
		if err != nil {
			return nil, err
		}
		expToken.Append(termToken)
		nestedToken.Append(expToken)

		_, err = processToken(tk, is(")"))
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(closeToken)

		termToken.Kind = "subroutineCall"
		return nestedToken, nil
	}

	if termToken.Type == tokenizer.INT_CONST {
		termToken.Kind = string(tokenizer.INT_CONST)
	}

	if termToken.Type == tokenizer.IDENTIFIER {
		subroutineSymbol, inSubroutineDec := subroutineSymbolTable[termToken.Raw]
		if inSubroutineDec {
			termToken.Var = subroutineSymbol
			termToken.Kind = subroutineSymbol.Kind
		}
		classSymbol, inClassDec := classSymbolTable[termToken.Raw]
		if inClassDec {
			termToken.Raw = fmt.Sprint(classSymbol.Index)
		}
	}

	// nestedToken.Append(termToken)
	nestedToken = termToken

	return nestedToken, nil
}

func CompileClassVarDec(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "classVarDec"}

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
		classSymbolTable[varNameToken.Raw] = &tokenizer.Var{
			Type:  typeToken.Raw,
			Kind:  classVarDecToken.Raw,
			Index: i,
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

func CompileVarDec(tk *tokenizer.Tokenizer, nvars *int) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "varDec", Kind: "varDec"}

	// varDecToken, err := processToken(tk, is("var"))
	_, err := processToken(tk, is("var"))
	if err != nil {
		return nil, err
	}
	// varDecToken.Kind = "varDec"
	// nestedToken.Append(varDecToken)

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}
	// nestedToken.Append(typeToken)

	for {
		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		varNameToken.Var = &tokenizer.Var{
			Index: *nvars,
			Type:  typeToken.Raw,
			Kind:  "var",
		}
		nestedToken.Append(varNameToken)
		subroutineSymbolTable[varNameToken.Raw] = varNameToken.Var
		*nvars++

		_, err = processToken(tk, is(","))
		if err != nil {
			break
		}
	}

	_, err = processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileSubroutine(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	subroutineSymbolTable = make(map[string]*tokenizer.Var)

	matcher := or(is("constructor"), is("function"), is("method"))
	if _, ok := matcher(tk.Current); !ok {
		return nil, notSubroutineDec
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
	nestedToken := subRoutineNameToken
	nestedToken.Kind = subRoutineDecToken.Raw
	nestedToken.Type = tokenizer.TokenType(subRoutineTypeToken.Kind)

	_, err = processToken(tk, is("("))
	if err != nil {
		return nil, err
	}

	paramsToken, err := CompileParameterList(tk)
	// _, err = CompileParameterList(tk)
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

	return nestedToken, nil
}

func CompileParameterList(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "parameterList"}

	for i := 0; ; i++ {
		if _, ok := isType()(tk.Current); !ok {
			break
		}

		typeToken, err := processToken(tk, isType())
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(typeToken)

		varNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		// nestedToken.Append(varNameToken)
		tvar := &tokenizer.Var{
			Type:  typeToken.Raw,
			Kind:  "arg",
			Index: i,
		}
		subroutineSymbolTable[varNameToken.Raw] = tvar

		varNameToken.Var = tvar
		nestedToken.Append(varNameToken)

		_, err = processToken(tk, is(","))
		if err != nil {
			break
		}
	}

	return nestedToken, nil
}

func CompileSubroutineBody(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "subroutineBody"}

	_, err := processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}

	var nvars int
	for varLine := 0; ; varLine++ {
		varToken, err := CompileVarDec(tk, &nvars)
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

	_, err = processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileStatements(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "statements"}

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
			nestedToken.Append(returnToken)
			continue
		}

		break
	}

	return nestedToken, nil
}

func CompileWhile(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	// nestedToken := tokenizer.MakeToken(&tokenizer.Token{Raw: "whileStatement"})

	whileToken, err := processToken(tk, is("while"))
	if err != nil {
		return nil, err
	}
	nestedToken := whileToken
	// nestedToken.Append(returnToken)

	_, err = processToken(tk, is("("))
	if err != nil {
		return nil, err
	}

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(expToken)

	_, err = processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}

	_, err = processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}

	statementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(statementToken)

	_, err = processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileReturn(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	returnToken, err := processToken(tk, is("return"))
	if err != nil {
		return nil, err
	}
	nestedToken := returnToken

	expToken, err := CompileExpression(tk)
	if err != nil && !errors.Is(err, notExpressionDec) {
		return nil, err
	}
	nestedToken.Append(expToken)

	_, err = processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileLet(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	let, err := processToken(tk, is("let"))
	if err != nil {
		return nil, err
	}

	termToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	sym, err := enforceVarDec(tk, termToken)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("LET SYMBOL \t %s => %+v\n", termToken, sym)

	termToken.Type = tokenizer.TokenType(sym.Type)
	termToken.Kind = sym.Kind
	// termToken.Kind = "varAssignment"
	// termToken.Raw = fmt.Sprint(sym.index)
	termToken.Var = sym

	if _, ok := is("[")(tk.Current); ok {
		_, err := processToken(tk, is("["))
		if err != nil {
			return nil, err
		}

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		let.Append(expToken)

		_, err = processToken(tk, is("]"))
		if err != nil {
			return nil, err
		}
	}

	_, err = processToken(tk, is("="))
	if err != nil {
		return nil, err
	}

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	let.Append(expToken)

	_, err = processToken(tk, is(";"))
	if err != nil {
		return nil, err
	}

	let.Append(termToken)

	return let, nil
}

func enforceVarDec(tk *tokenizer.Tokenizer, termToken *tokenizer.Token) (*tokenizer.Var, error) {
	subroutineSymbol, inSubroutineDec := subroutineSymbolTable[termToken.Raw]
	classSymbol, inClassDec := classSymbolTable[termToken.Raw]
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

func CompileIf(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	// nestedToken := tokenizer.MakeToken(&tokenizer.Token{Raw: "ifStatement"})

	ifToken, err := processToken(tk, is("if"))
	if err != nil {
		return nil, err
	}
	nestedToken := ifToken
	// nestedToken.Append(ifToken)

	_, err = processToken(tk, is("("))
	if err != nil {
		return nil, err
	}

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(expToken)

	_, err = processToken(tk, is(")"))
	if err != nil {
		return nil, err
	}

	_, err = processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}

	statementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(statementToken)

	_, err = processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	_, hasElse := is("else")(tk.Current)
	if !hasElse {
		return nestedToken, nil
	}

	elseToken, err := processToken(tk, is("else"))
	if err != nil {
		return nil, err
	}
	nestedToken.Append(elseToken)

	_, err = processToken(tk, is("{"))
	if err != nil {
		return nil, err
	}

	elseStatementToken, err := CompileStatements(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(elseStatementToken)

	_, err = processToken(tk, is("}"))
	if err != nil {
		return nil, err
	}

	return nestedToken, nil
}

func CompileDo(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	do, err := processToken(tk, is("do"))
	if err != nil {
		return nil, err
	}

	varClassNameToken, err := processToken(tk, isIdentifier())
	if err != nil {
		return nil, err
	}
	subroutineCall := varClassNameToken
	subroutineCall.Kind = "subroutineCall"

	if _, ok := is(".")(tk.Current); ok {
		dotToken, err := processToken(tk, is("."))
		if err != nil {
			return nil, err
		}
		subroutineCall.Raw += dotToken.Raw

		subroutineNameToken, err := processToken(tk, isIdentifier())
		if err != nil {
			return nil, err
		}
		subroutineCall.Raw += subroutineNameToken.Raw
	}

	_, err = processToken(tk, is("("))
	if err != nil {
		return nil, err
	}

	expListToken, err := CompileExpressionList(tk)
	if err != nil {
		return nil, err
	}
	expListToken.Append(subroutineCall)
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

func CompileExpressionList(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "expressionList"}

	if _, ok := is(")")(tk.Current); ok {
		return nestedToken, nil
	}

	for {
		expToken, err := CompileExpression(tk)
		if errors.Is(err, notExpressionDec) {
			break
		}
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		if _, ok := is(",")(tk.Current); !ok {
			break
		}
		_, err = processToken(tk, is(","))
		if err != nil {
			return nil, err
		}
	}

	return nestedToken, nil
}

func CompileExpression(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "expression"}

	if _, ok := is(";")(tk.Current); ok {
		return nil, notExpressionDec
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
	onerror.Log(err)
	return token
}
