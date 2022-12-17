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

	vmBuf := vm.New(out)
	compiler := engine.New(vmBuf)

	if err := compiler.Class(&tk); err != nil {
		return err
	}

	// compiled, err := engine.CompileClass(&tk)
	// if err != nil {
	// 	return err
	// }

	// vm.Output(compiled, out)

	return nil
}
