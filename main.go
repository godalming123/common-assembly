package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Parsing main.ca into a list of keywords...")
	parsedCode := convertFileIntoParsedCode("basic.ca")
	if len(parsedCode.parsingErrors) > 0 {
		fmt.Println(" Line   Column   Error message from parsing to keywords")
		fmt.Println(" ────   ──────   ──────────────────────────────────────")
	}
	for _, err := range parsedCode.parsingErrors {
		fmt.Printf(" %4d   %6d   %s\n", err.line, err.column, err.msg)
	}

	fmt.Println("Parsing keywords into assembly...")
	parsingState := keywordParsingState{
		currentKeyword:               0,
		numberOfControlFlowJumpNames: 0,
		targetArchitecture:           amd64,
		targetOS:                     linux,
		keywords:                     parsedCode.keywords,
		inlineValues:                 []string{},
		dataSectionText:              "",
		functionStack:                []functionProperties{},
	}
	err := parsingState.parseFromBeginneng()
	if err.msg != nil {
		fmt.Println(" Line   Column   Error message from parsing to keywords")
		fmt.Println(" ────   ──────   ──────────────────────────────────────")
		fmt.Printf(" %4d   %6d   %s\n", err.line, err.column, err.msg)
	}

	fmt.Println("Writing assembly to main.asm...")
	os.WriteFile("main.asm", []byte(parsingState.dataSectionText), 0644)
}
