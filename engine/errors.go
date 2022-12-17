package engine

import "errors"

var (
	notClassVarDec   = errors.New("not a class variable declaration")
	notLocalVarDec   = errors.New("not a local variable declaration")
	notSubroutineDec = errors.New("not a subroutine declaration")
	notExpressionDec = errors.New("not an expression declaration")
)
