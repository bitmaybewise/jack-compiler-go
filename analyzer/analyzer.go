package analyzer

import (
	"os"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/engine"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
	"github.com/hlmerscher/jack-compiler-go/writer"
)

func Compile(file *os.File, out *strings.Builder) error {
	tk := tokenizer.New(file)

	_, err := tk.Advance()
	if err != nil {
		return err
	}

	compiled, err := engine.CompileClass(&tk)
	if err != nil {
		return err
	}

	err = writer.Output(out, compiled)
	if err != nil {
		return err
	}

	return nil
}
