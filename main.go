package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hlmerscher/jack-compiler-go/analyzer"
	"github.com/hlmerscher/jack-compiler-go/logger"
)

func main() {
	var filename, dirname string
	var verbose bool
	flag.StringVar(&filename, "f", "", "the filename of the vm source file")
	flag.StringVar(&dirname, "d", "", "the directory of the vm source files")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()
	if filename == "" && dirname == "" {
		panic("filename/directory is missing")
	}
	logger.Toggle(verbose)

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
		logger.Error(err)
	}
	writeToFile(filename, out.String())
}

func openJackFile(filename string) *os.File {
	inputFile, err := os.Open(filename)
	logger.Errorf("error opening file\n", err)
	return inputFile
}

func dirFilenames(dirname string) []string {
	entries, err := os.ReadDir(dirname)
	logger.Errorf("error reading directory\n", err)

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
	logger.Error(err)
}
