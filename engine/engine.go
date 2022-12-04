package engine

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hlmerscher/jack-compiler-go/logger"
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
	classSymbolTable["this"] = &tokenizer.Var{
		Index: 0,
		Type:  classNameToken.Raw,
		Kind:  "class",
	}

	processTokenOrPanics(tk, is("{"))

	var nvars int
	for {
		varDec, err := CompileClassVarDec(tk, &nvars)
		if errors.Is(err, notClassVarDec) {
			break
		}
		if varDec.Kind == "field" {
			for _, child := range varDec.Children() {
				if child.Type == "identifier" {
					class.NFields++
				}
			}
		}
		logger.Error(err)
	}

	for {
		subRoutineToken, err := CompileSubroutine(tk)
		if errors.Is(err, notSubroutineDec) {
			break
		}
		logger.Error(err)

		class.Append(subRoutineToken)
	}

	processTokenOrPanics(tk, is("}"))

	return class, nil
}

func CompileTerm(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "term"}

	// unaryOp term
	if _, ok := isUnaryOp()(tk.Current); ok {
		opToken := processTokenOrPanics(tk, isUnaryOp())
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
		processTokenOrPanics(tk, is("("))

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		nestedToken.Append(expToken)

		processTokenOrPanics(tk, is(")"))

		return nestedToken, nil
	}

	// varName
	termToken := processTokenOrPanics(tk, isTerm())
	_var, err := enforceVarDec(tk, termToken)
	if err != nil {
		return nil, err
	}

	// varName[expression]
	if _, ok := is("[")(tk.Current); ok {
		processTokenOrPanics(tk, is("["))

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		// arrayIndex := expToken.Children()[0]
		// termToken.ArrayIndex = arrayIndex
		for _, child := range expToken.Children() {
			termToken.Append(child)
		}
		termToken.Var = _var
		termToken.Kind = _var.Kind
		nestedToken.Append(termToken)

		processTokenOrPanics(tk, is("]"))

		return nestedToken, nil
	}

	// subroutineCall
	if _, ok := is(".")(tk.Current); ok {
		if _var != nil {
			termToken.Method = _var
			nestedToken.Append(
				&tokenizer.Token{Raw: "var", Type: tokenizer.IDENTIFIER, Kind: "var", Var: _var},
			)
		} else {
			termToken.Constructor = &tokenizer.Var{
				Type: termToken.Kind,
				Kind: "constructor",
			}
		}
		termToken.Kind = "subroutineCall"

		processTokenOrPanics(tk, is("."))

		subroutineNameToken := processTokenOrPanics(tk, isIdentifier())
		termToken.Raw = subroutineNameToken.Raw

		processTokenOrPanics(tk, is("("))

		expToken, err := CompileExpressionList(tk)
		if err != nil {
			return nil, err
		}
		expToken.Append(termToken)
		nestedToken.Append(expToken)

		processTokenOrPanics(tk, is(")"))

		return nestedToken, nil
	}

	if termToken.Type == tokenizer.IDENTIFIER {
		termToken.Var = _var
		termToken.Kind = "var"
	}

	if termToken.Type == tokenizer.INT_CONST {
		termToken.Kind = string(tokenizer.INT_CONST)
		// idx, _ := strconv.Atoi(termToken.Raw)
		// termToken.Var = &tokenizer.Var{Kind: "constant", Index: idx, Type: termToken.Kind}
	}

	return termToken, nil
}

func CompileClassVarDec(tk *tokenizer.Tokenizer, nvars *int) (*tokenizer.Token, error) {
	matcher := or(is("static"), is("field"))
	if _, ok := matcher(tk.Current); !ok {
		return nil, notClassVarDec
	}

	classVarDecToken, err := processToken(tk, matcher)
	if err != nil {
		return nil, err
	}
	nestedToken := classVarDecToken
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
			Index: *nvars,
		}
		*nvars++

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

	_, err := processToken(tk, is("var"))
	if err != nil {
		return nil, err
	}

	typeToken, err := processToken(tk, isType())
	if err != nil {
		return nil, err
	}

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

	subRoutineDecToken := processTokenOrPanics(tk, matcher)
	subRoutineTypeToken := processTokenOrPanics(tk, is("void"), isType())
	subRoutineNameToken := processTokenOrPanics(tk, isIdentifier())

	var args int
	if subRoutineDecToken.Kind == "method" {
		subRoutineNameToken.Var = &tokenizer.Var{
			Index: 0,
			Kind:  "arg",
			Type:  classSymbolTable["this"].Type,
		}
		args++
	}

	nestedToken := subRoutineNameToken
	nestedToken.Kind = subRoutineDecToken.Raw
	nestedToken.Type = tokenizer.TokenType(subRoutineTypeToken.Kind)

	processTokenOrPanics(tk, is("("))

	paramsToken := CompileParameterList(tk, &args)
	nestedToken.Append(paramsToken)

	processTokenOrPanics(tk, is(")"))

	bodyToken, err := CompileSubroutineBody(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.Append(bodyToken)

	return nestedToken, nil
}

func CompileParameterList(tk *tokenizer.Tokenizer, args *int) *tokenizer.Token {
	nestedToken := &tokenizer.Token{Raw: "parameterList"}

	for i := *args; ; i++ {
		if _, ok := isType()(tk.Current); !ok {
			break
		}

		typeToken := processTokenOrPanics(tk, isType())

		varNameToken := processTokenOrPanics(tk, isIdentifier())
		tvar := &tokenizer.Var{
			Type:  typeToken.Raw,
			Kind:  "arg",
			Index: i,
		}
		subroutineSymbolTable[varNameToken.Raw] = tvar

		varNameToken.Var = tvar
		nestedToken.Append(varNameToken)

		_, err := processToken(tk, is(","))
		if err != nil {
			break
		}
	}

	return nestedToken
}

func CompileSubroutineBody(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	nestedToken := &tokenizer.Token{Raw: "subroutineBody"}

	processTokenOrPanics(tk, is("{"))

	var nvars int
	for varLine := 0; ; varLine++ {
		varToken, err := CompileVarDec(tk, &nvars)
		if err != nil {
			break
		}
		nestedToken.Append(varToken)
	}

	statementsToken, _ := CompileStatements(tk)
	nestedToken.Append(statementsToken)

	processTokenOrPanics(tk, is("}"))

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
	let := processTokenOrPanics(tk, is("let"))

	termToken := processTokenOrPanics(tk, isIdentifier())
	sym, err := enforceVarDec(tk, termToken)
	if err != nil {
		return nil, err
	}

	termToken.Type = tokenizer.TokenType(sym.Type)
	termToken.Kind = sym.Kind
	termToken.Var = sym

	if _, ok := is("[")(tk.Current); ok {
		processTokenOrPanics(tk, is("["))

		expToken, err := CompileExpression(tk)
		if err != nil {
			return nil, err
		}
		// termToken.ArrayIndex = expToken.Children()[0]
		for _, child := range expToken.Children() {
			termToken.Append(child)
		}

		processTokenOrPanics(tk, is("]"))
	}

	processTokenOrPanics(tk, is("="))

	expToken, err := CompileExpression(tk)
	if err != nil {
		return nil, err
	}
	let.Append(expToken)

	processTokenOrPanics(tk, is(";"))

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
	do := processTokenOrPanics(tk, is("do"))

	varClassNameToken := processTokenOrPanics(tk, isIdentifier())
	subroutineCall := varClassNameToken
	subroutineCall.Kind = "subroutineCall"

	startsWithLowercase, _ := regexp.MatchString("^[a-z]+", subroutineCall.Raw)

	_var, err := enforceVarDec(tk, subroutineCall)
	if err == nil && _var != nil {
		subroutineCall.Method = _var
	} else if startsWithLowercase {
		subroutineCall.Method = classSymbolTable["this"]
	}

	if subroutineCall.Method != nil {
		do.Append(&tokenizer.Token{Kind: "var", Var: subroutineCall.Method})
	}

	if _, ok := is(".")(tk.Current); ok {
		dotToken := processTokenOrPanics(tk, is("."))
		subroutineCall.Raw += dotToken.Raw

		subroutineNameToken := processTokenOrPanics(tk, isIdentifier())
		subroutineCall.Raw += subroutineNameToken.Raw
		if _var != nil {
			subroutineCall.Raw = subroutineNameToken.Raw
		}
	}

	processTokenOrPanics(tk, is("("))

	expListToken, err := CompileExpressionList(tk)
	if err != nil {
		return nil, err
	}
	expListToken.Append(subroutineCall)
	do.Append(expListToken)

	processTokenOrPanics(tk, is(")"))
	processTokenOrPanics(tk, is(";"))

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
		processTokenOrPanics(tk, is(","))
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

		opToken := processTokenOrPanics(tk, isOp())

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
	logger.Error(err)
	return token
}
