package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("Reading the text in basic.ca...")
	rawText, err := os.ReadFile("basic.ca")
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Lexing basic.ca into a list of keywords...")
	parsedCode := lexCode(string(rawText))
	printErrorsInCode("basic.ca", strings.Split(string(rawText), "\n"), parsedCode.parsingErrors)

	fmt.Println("Parsing keywords into abstract syntax tree...")
	AST, error := keywordsToAST(parsedCode.keywords)
	if error.msg != nil {
		printErrorsInCode("basic.ca", strings.Split(string(rawText), "\n"), []codeParsingError{
			error,
		})
	}

	for _, ASTitem := range AST {
		ASTitem.print(0)
	}

	// fmt.Println("Writing assembly to main.asm...")
	// os.WriteFile("main.asm", []byte(parsingState.dataSectionText), 0644)
}
