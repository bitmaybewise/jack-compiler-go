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
	if _, err := tk.Advance(); err != nil {
		return err
	}

	vmBuf := vm.New(out)
	compiler := engine.New(vmBuf)
	if err := compiler.Class(&tk); err != nil {
		return err
	}

	return nil
}
