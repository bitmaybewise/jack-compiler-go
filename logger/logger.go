package logger

import (
	"fmt"
	"log"
)

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

func Error(err error) {
	Errorf("", err)
}

func Errorf(msg string, err any) {
	if err != nil {
		log.Fatalf("\n%s%s", msg, err)
	}
}
