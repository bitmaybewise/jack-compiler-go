package analyzer

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/tokenizer"
)

type tokensWrapper struct {
	XMLName xml.Name `xml:"tokens"`
	Tokens  []tokenizer.Token
}

func Compile(file *os.File, out *strings.Builder) {
	tk := tokenizer.New(file)

	tw := tokensWrapper{Tokens: make([]tokenizer.Token, 0)}
	for {
		token, err := tk.Advance()
		if errors.Is(err, io.EOF) {
			break
		}

		tw.Tokens = append(tw.Tokens, token)
	}

	result, err := xml.MarshalIndent(tw, "", " ")
	if err != nil {
		panic(err)
	}
	out.Write(result)
}
