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

		lineStartsWithQuote := tk.tokenizedLine[0] == '"'
		symbol := isSymbol(string(char))

		if char == ' ' && !lineStartsWithQuote {
			break
		}
		if char == '"' && lineStartsWithQuote && i > 0 {
			currentIndex++
			rawToken.WriteRune(char)
			break
		}

		if symbol && !lineStartsWithQuote && rawToken.Len() > 0 {
			break
		}

		if symbol && !lineStartsWithQuote && rawToken.Len() == 0 {
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

type Var struct {
	Index int
	Type  string
	Kind  string
}

func (v *Var) String() string {
	return fmt.Sprintf("{index:%d type:%s kind:%s}", v.Index, v.Type, v.Kind)
}

type Token struct {
	Raw  string
	Type TokenType

	// nested token
	Kind     string
	Parent   *Token
	children []*Token
	NFields  int

	Var         *Var
	Method      *Var
	Constructor *Var
	ArrayIndex  *Token
}

func (t *Token) String() string {
	var s []string
	s = []string{fmt.Sprintf("%s:%s", t.Type, t.Raw)}

	if t.Kind != "" {
		s = append(s, fmt.Sprintf("kind:%s", t.Kind))
	}
	if t.Var != nil {
		s = append(s, fmt.Sprintf("var:%s", t.Var))
	}
	if t.Method != nil {
		s = append(s, fmt.Sprintf("method:%s", t.Method))
	}
	if t.Constructor != nil {
		s = append(s, fmt.Sprintf("constructor:%s", t.Constructor))
	}
	if t.Kind == "class" {
		s = append(s, fmt.Sprintf("nFields:%d", t.NFields))
	}
	if t.ArrayIndex != nil {
		s = append(s, fmt.Sprintf("array: %s", t.ArrayIndex))
	}

	return fmt.Sprintf("(%s)", strings.Join(s, " "))
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

func (t *Token) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
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

func (t *Token) NLocalVars() int {
	var count int
	for _, child := range t.Children() {
		if child.Kind == "varDec" {
			count += len(child.Children())
		}
		count += child.NLocalVars()
	}
	return count
}

func (t *Token) NStackVars() int {
	if len(t.Parent.children) == 0 {
		return 0
	}
	return len(t.Parent.children) - 1
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
