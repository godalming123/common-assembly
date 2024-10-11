package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	fileName := "main.ca"

	fmt.Println("Reading the text in " + fileName + "...")
	rawText, err := os.ReadFile(fileName)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	println("Lexing into a list of keywords...")
	parsedCode := lexCode(string(rawText))
	if printErrorsInCode(fileName, strings.Split(string(rawText), "\n"), parsedCode.parsingErrors) {
		os.Exit(1)
	}

	println("Parsing keywords into abstract syntax tree...")
	AST, error := keywordsToAST(parsedCode.keywords)
	if error.msg != nil {
		printErrorsInCode(fileName, strings.Split(string(rawText), "\n"), []codeParsingError{error})
		os.Exit(1)
	}

	println("Compiling abstract syntax tree into assembly...")
	dataSectionText, textSectionAssembly, error := compileAssembly(AST)
	if error.msg != nil {
		printErrorsInCode(fileName, strings.Split(string(rawText), "\n"), []codeParsingError{error})
		os.Exit(1)
	}

	fmt.Println("Writing assembly to out.asm...")
	err = os.WriteFile("out.asm", []byte(".global _start\n.text"+textSectionAssembly+dataSectionText+"\n"), 0644)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Assembling assembly to out.o...")
	out, err := exec.Command("as", "out.asm", "-o", "out.o").CombinedOutput()
	print(string(out))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Linking out.o to out...")
	out, err = exec.Command("ld", "out.o", "-o", "out").CombinedOutput()
	print(string(out))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
