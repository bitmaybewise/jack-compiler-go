package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/analyzer"
)

func main() {
	var filename, dirname string
	flag.StringVar(&filename, "f", "", "the filename of the vm source file")
	flag.StringVar(&dirname, "d", "", "the directory of the vm source files")
	flag.Parse()
	if filename == "" && dirname == "" {
		panic("filename/directory is missing")
	}

	if filename != "" {
		analyzeFile(filename)
	}
	if dirname != "" {
		dirname = strings.TrimSuffix(dirname, "/")
		for _, filename := range dirFilenames(dirname) {
			analyzeFile(filename)
		}
	}
}

func analyzeFile(filename string) {
	fmt.Printf("input:\t%s\n", filename)

	sourceFile := openJackFile(filename)
	defer sourceFile.Close()

	out := new(strings.Builder)
	err := analyzer.Compile(sourceFile, out)
	if err != nil {
		onError(err)
	}
	writeToFile(filename, out.String())
}

func openJackFile(filename string) *os.File {
	inputFile, err := os.Open(filename)
	onErrorf("error opening file\n", err)
	return inputFile
}

func dirFilenames(dirname string) []string {
	entries, err := os.ReadDir(dirname)
	onErrorf("error reading directory\n", err)

	filenames := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), "jack") {
			name := dirname + "/" + entry.Name()
			filenames = append(filenames, name)
		}
	}

	return filenames
}

func writeToFile(filename string, content string) {
	outputFilename := strings.Replace(filename, ".jack", ".vm", 1)
	fmt.Printf("output:\t%s\n", outputFilename)

	err := os.WriteFile(outputFilename, []byte(content), 0666)
	onError(err)
}

func onError(err error) {
	onErrorf("", err)
}

func onErrorf(msg string, err error) {
	if err != nil {
		log.Fatalf("\n%s%s", msg, err)
	}
}
