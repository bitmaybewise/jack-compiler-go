package logger

import "fmt"

var verbose = false

func Toggle(flag bool) {
	verbose = flag
}

func Print(values ...any) {
	if !verbose {
		return
	}

	fmt.Print(values...)
}

func Printf(format string, values ...any) {
	if !verbose {
		return
	}

	fmt.Printf(format, values...)
}

func Println(values ...any) {
	if !verbose {
		return
	}

	fmt.Println(values...)
}
