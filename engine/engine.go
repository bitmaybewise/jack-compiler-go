package engine

import (
	"encoding/xml"
	"fmt"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

type NestedToken struct {
	XMLName  xml.Name
	Children []xml.Marshaler
}

func (nt *NestedToken) append(token xml.Marshaler) {
	if token != nil {
		nt.Children = append(nt.Children, token)
	}
}

func makeNestedToken(token *tokenizer.Token) *NestedToken {
	return &NestedToken{
		XMLName:  xml.Name{Local: token.Raw},
		Children: []xml.Marshaler{token},
	}
}

func CompileClass(tk *tokenizer.Tokenizer) (*NestedToken, error) {
	token, err := processToken(tk, "class")
	if err != nil {
		return nil, err
	}
	nestedToken := makeNestedToken(token)

	termToken, err := CompileTerm(tk)
	if err != nil {
		return nil, err
	}
	nestedToken.append(termToken)

	// varDecToken, err := CompileClassVarDec(tk)
	// if err != nil {
	// 	return nil, err
	// }
	// nestedToken.append(varDecToken)

	// subRoutineToken, err := CompileSubroutine(tk)
	// if err != nil {
	// 	return nil, err
	// }
	// nestedToken.append(subRoutineToken)

	openToken, err := processToken(tk, "{")
	if err != nil {
		return nil, err
	}
	nestedToken.append(openToken)

	closeToken, err := processToken(tk, "}")
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

func CompileClassVarDec(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	return nil, fmt.Errorf("not implemented error")
}

func CompileSubroutine(tk *tokenizer.Tokenizer) (*tokenizer.Token, error) {
	return nil, fmt.Errorf("not implemented error")
}

func processToken(tk *tokenizer.Tokenizer, expectedToken string) (*tokenizer.Token, error) {
	if tk.Current.Raw != expectedToken {
		return nil, fmt.Errorf("wrong token error, expected <%s>, got <%s>", expectedToken, tk.Current.Raw)
	}
	token := tk.Current

	_, err := tk.Advance()
	if err != nil {
		return nil, err
	}

	return &token, nil
}
