package analyzer

import (
	"os"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/engine"
	"github.com/hlmerscher/jack-compiler-go/tokenizer"
	"github.com/hlmerscher/jack-compiler-go/vm"
)

func Compile(file *os.File, out *strings.Builder) error {
	tk := tokenizer.New(file)

	_, err := tk.Advance()
	if err != nil {
		return err
	}

	writer := vm.NewWriter(out)

	_, err = engine.CompileClass(&tk)
	if err != nil {
		return err
	}

	_, err = writer.Output()
	if err != nil {
		return err
	}

	return nil
}
