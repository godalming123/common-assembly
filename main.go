package main

import "fmt"

func main() {
	parsedCode := convertFileIntoParsedCode("main.TODO")
	fmt.Println(" Line   Column   Message")
	fmt.Println(" ────   ──────   ───────")
	for _, err := range parsedCode.parsingErrors {
		fmt.Printf(" %4d   %6d   Error: %s\n", err.line, err.column, err.errorMsg)
	}
	for _, keyword := range parsedCode.keywords {
		fmt.Printf(
			" %4d   %6d   Keyword of type %-26s: `%s`\n",
			keyword.line,
			keyword.column,
			convertKeywordTypeToString(keyword.keywordType),
			keyword.contents,
		)
	}

	//	err := os.WriteFile("assembly.s", []byte(
	//		fmt.Sprintf(`
	//
	// .global _start
	// .intel_syntax noprefix
	//
	// _start:
	//
	//	 %s`,
	//				"HEllo",
	//			),
	//		), 0644)
	//		if err != nil {
	//			log.Fatal(err)
	//		}
}
