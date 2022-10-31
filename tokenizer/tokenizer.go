package tokenizer

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

var (
	Ignored    = errors.New("ignored line")
	EmptyToken = Token{}
)

func New(input io.Reader) Tokenizer {
	return Tokenizer{
		input:   bufio.NewReader(input),
		Current: EmptyToken,
	}
}

type Tokenizer struct {
	input         *bufio.Reader
	CurrentLine   string
	tokenizedLine string
	LineNr        int
	Current       Token
}

func (tk *Tokenizer) HasMoreTokens() bool {
	return false
}

func (tk *Tokenizer) Advance() (Token, error) {
	if len(tk.tokenizedLine) > 0 {
		return tk.nextToken(), nil
	}

	tokenizedLine, err := tk.ReadLine()
	tk.CurrentLine = tokenizedLine
	if errors.Is(err, Ignored) {
		return tk.Advance()
	}
	if errors.Is(err, io.EOF) {
		return tk.Current, nil
	}
	if err != nil {
		return EmptyToken, err
	}
	tk.tokenizedLine = tokenizedLine

	return tk.Advance()
}

func (tk *Tokenizer) nextToken() Token {
	line := strings.Trim(tk.tokenizedLine, " ")
	if line == "" {
		return EmptyToken
	}

	var rawToken strings.Builder

	var currentIndex int
	for i, char := range tk.tokenizedLine {
		currentIndex = i
		if char == ' ' && tk.tokenizedLine[0] != '"' {
			break
		}
		if isSymbol(string(char)) && rawToken.Len() > 0 {
			break
		}
		if isSymbol(string(char)) && rawToken.Len() == 0 {
			currentIndex++
			rawToken.WriteRune(char)
			break
		}
		rawToken.WriteRune(char)
	}
	tk.tokenizedLine = strings.Trim(line[currentIndex:], " ")

	tk.Current = Token{
		Raw:  rawToken.String(),
		Type: parseTokenType(rawToken.String()),
		Kind: rawToken.String(),
	}

	return tk.Current
}

func (tk *Tokenizer) ReadLine() (string, error) {
	line, err := nextLine(tk.input)
	if err != nil {
		return "", err
	}
	tk.LineNr++

	if isSingleLineComment(line) {
		return "", Ignored
	}
	if isMultiLineComment(line) {
		for {
			line, err = nextLine(tk.input)
			if err != nil {
				return "", err
			}
			tk.LineNr++

			if isEndOfMultiLineComment(line) {
				return "", Ignored
			}
		}
	}
	if commentFoundAt, ok := hasCommentAtTheEnd(line); ok {
		line = line[:commentFoundAt-1]
	}

	line = strings.Trim(line, " ")
	if line == "" {
		return "", Ignored
	}

	return line, nil
}

func nextLine(input *bufio.Reader) (string, error) {
	line, err := input.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, "\n", "")
	line = strings.ReplaceAll(line, "\t", "")
	line = strings.Trim(line, " ")
	return line, nil
}

func isSingleLineComment(line string) bool {
	return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") && strings.HasSuffix(line, "*/")
}

func isMultiLineComment(line string) bool {
	return strings.HasPrefix(line, "/*") && !strings.HasSuffix(line, "*/")
}

func isEndOfMultiLineComment(line string) bool {
	return strings.HasSuffix(line, "*/")
}

func hasCommentAtTheEnd(line string) (int, bool) {
	commentFoundAt := strings.Index(line, "//")
	return commentFoundAt, commentFoundAt > 1
}

type TokenType string

const (
	KEYWORD      = TokenType("keyword")
	SYMBOL       = TokenType("symbol")
	IDENTIFIER   = TokenType("identifier")
	INT_CONST    = TokenType("integerConstant")
	STRING_CONST = TokenType("stringConstant")
	UNKNOWN      = TokenType("UNKNOWN")
)

type Token struct {
	Raw  string
	Type TokenType

	// nested token
	Kind     string
	Parent   *Token
	children []*Token
}

func (nt *Token) Append(token *Token) {
	if token == nil {
		return
	}

	nt.children = append(nt.children, token)
	token.Parent = nt
}

func (nt *Token) Children() []*Token {
	return nt.children
}

func (t Token) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = string(t.Type)
	value := strings.Trim(t.Raw, "\"")
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")
	err := e.EncodeElement(fmt.Sprintf(" %s ", value), start)
	if err != nil {
		return err
	}
	return nil
}

func parseTokenType(value string) TokenType {
	switch {
	case isKeyword(value):
		return KEYWORD
	case isSymbol(value):
		return SYMBOL
	case isInteger(value):
		return INT_CONST
	case isString(value):
		return STRING_CONST
	case isIdentifier(value):
		return IDENTIFIER
	}
	return UNKNOWN
}

var keywords = []string{
	"class",
	"constructor",
	"function",
	"method",
	"field",
	"static",
	"var",
	"int",
	"char",
	"boolean",
	"void",
	"true",
	"false",
	"null",
	"this",
	"let",
	"do",
	"if",
	"else",
	"while",
	"return",
}

func isKeyword(value string) bool {
	return slices.Contains(keywords, value)
}

var symbols = []string{
	"{", "}",
	"(", ")",
	"[", "]",
	".", ",", ";",
	"+", "-", "*", "/",
	"&", "|",
	"<", ">",
	"=", "~",
}

func isSymbol(value string) bool {
	return slices.Contains(symbols, value)
}

func isInteger(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func isString(value string) bool {
	return strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")
}

func isIdentifier(value string) bool {
	return regexp.MustCompile(`^\D[a-zA-Z0-9_]*`).Match([]byte(value))
}
