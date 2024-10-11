package main

// Lexer.go
// ========
// Responsible for parsing common assembly into a list of keywords and errors
// that are encountered during the process.

import (
	"errors"
	"fmt"
	"strings"
)

///////////////////////
// TEXT AND POSITION //
///////////////////////

// Stores a position within a piece of text
type textAndPosition struct {
	text     string
	index    int
	location textLocation
}

// If `text.index` is already at the end of `text.text`, then the function
// returns true. Otherwise the function increases `text.index` by one, while
// handling newlines so that `text.line` and `text.column` are still valid
// given `text.index`s new value.
func (text *textAndPosition) moveForward() bool {
	if text.index >= len(text.text)-1 {
		return true
	}
	if text.text[text.index] == '\n' {
		text.location.column = 1
		text.location.line += 1
	} else {
		text.location.column += 1
	}
	text.index += 1
	return false
}

// Runs the `checker` function with the current byte parsed as an argument and
// then the `moveForward` function until either function returns true. If the
// `checker` function returned true, then this function returns true. If the
// `moveForward` function returned true, then this function returns false.
func (text *textAndPosition) findUntil(checker func(byte) bool) bool {
	for true {
		if checker(text.text[text.index]) {
			return true
		}
		if text.moveForward() {
			return false
		}
	}
	panic("Unreachable")
}

// Same as `findUntil`, but returns the string that was iterated over instead
// of true or false.
func (text *textAndPosition) findUntilWithIteratedString(checker func(byte) bool) string {
	start := text.index
	text.findUntil(checker)
	return text.text[start:text.index]
}

//////////////////
// KEYWORD TYPE //
//////////////////

//go:generate stringer -type=keywordType
type keywordType uint8

const (
	Name                       keywordType = iota // myVariableName3, _
	Mutatability                                  // mut, arg, mutArg
	Register                                      // sys0, sys1, sys2, ...
	StringValue                                   // "Foo", "Bar"
	BoolValue                                     // true, false
	IntNumber                                     // 4, 23
	FloatNumber                                   // 4.20, 6.9
	IncreaseNesting                               // (, {, [
	DecreaseNesting                               // ), }, ]
	VariableCreationSyntax                        // ::
	VariableModificationSyntax                    // =, ++, --, +=, -=, *=, /=, %=
	ControlFlowSyntax                             // while, if, else, elif
	ComparisonSyntax                              // ==, !=, >, <, >=, <=, or, and
	ListSyntax                                    // ,
	AntiListSyntax                                // ...
	IterationSyntax                               // ..
	Syscall                                       // syscall
	Import                                        // import
	FunctionReturn                                // return
	PointerOf                                     // ^
	Comment                                       // # My comment 2
	Newline                                       // \n
	UnknownKeywordType
)

type keyword struct {
	contents    string
	keywordType keywordType
	nesting     uint8
	location    textLocation
}

// Displays a slice of keywords in a table
func printKeywords(keywords []keyword) {
	longestLine := 4
	longestColumn := 6
	longestNesting := 7
	longestType := 12
	longestContents := 16
	for _, keyword := range keywords {
		if len(fmt.Sprint(keyword.location.line)) > longestLine {
			longestLine = len(fmt.Sprint(keyword.location.line))
		}
		if len(fmt.Sprint(keyword.location.column)) > longestColumn {
			longestColumn = len(fmt.Sprint(keyword.location.column))
		}
		if len(fmt.Sprint(int(keyword.nesting))) > longestNesting {
			longestNesting = len(fmt.Sprint(keyword.nesting))
		}
		if len(keyword.keywordType.String()) > longestType {
			longestType = len(keyword.keywordType.String())
		}
		if len(strings.Replace(keyword.contents, "\n", "\\n", -1)) > longestContents {
			longestContents = len(strings.Replace(keyword.contents, "\n", "\\n", -1))
		}
	}
	printTableSymbolsRow(
		topLeftQuarterCircle, horizontalLine, downTriad, topRightQuarterCircle,
		longestLine, longestColumn, longestNesting, longestType, longestContents,
	)
	printTableRow([]tableCell{
		{contents: "Line", width: longestLine},
		{contents: "Column", width: longestColumn},
		{contents: "Nesting", width: longestNesting},
		{contents: "Keyword type", width: longestType},
		{contents: "Keyword contents", width: longestContents},
	})
	printTableSymbolsRow(
		rightTriad, horizontalLine, crossingLines, leftTriad,
		longestLine, longestColumn, longestNesting, longestType, longestContents,
	)
	for _, keyword := range keywords {
		printTableRow([]tableCell{
			{contents: fmt.Sprint(keyword.location.line), width: longestLine, rightAligned: true},
			{contents: fmt.Sprint(keyword.location.column), width: longestColumn, rightAligned: true},
			{contents: fmt.Sprint(keyword.nesting), width: longestNesting, rightAligned: true},
			{contents: keyword.keywordType.String(), width: longestType},
			{contents: strings.Replace(keyword.contents, "\n", "\\n", -1), width: longestContents},
		})
	}
	printTableSymbolsRow(
		bottomLeftQuarterCircle, horizontalLine, upTriad, bottomRightQuarterCircle,
		longestLine, longestColumn, longestNesting, longestType, longestContents,
	)
}

/////////////////////////////
// CODE PARSING ERROR TYPE //
/////////////////////////////

type codeParsingError struct {
	msg      error
	location textLocation
}

//////////////////////
// PARSED CODE TYPE //
//////////////////////

type parsedCode struct {
	keywords      []keyword
	parsingErrors []codeParsingError
}

////////////////////////
// OTHER  HELPER CODE //
////////////////////////

func isNotIgnoreableWhitespace(charecter byte) bool {
	if charecter == ' ' || charecter == '\t' {
		return false
	}
	return true
}

///////////////
// MAIN CODE //
///////////////

func numberToKeyword(text *textAndPosition) (keywordType, string) {
	// Parse any digits (and `_`) into keywordContents
	keywordType := IntNumber
	keywordContents := text.findUntilWithIteratedString(func(charecter byte) bool {
		if ('0' <= charecter && charecter <= '9') || charecter == '_' {
			return false
		}
		return true
	})
	text.findUntil(isNotIgnoreableWhitespace)

	// Handle if the number if a float
	if text.index < len(text.text)-1 &&
		text.text[text.index] == '.' &&
		text.text[text.index+1] != '.' {
		keywordType = FloatNumber
		keywordContents += "."
		text.index++
		text.findUntil(isNotIgnoreableWhitespace)
		keywordContents += text.findUntilWithIteratedString(func(charecter byte) bool {
			if ('0' <= charecter && charecter <= '9') || charecter == '_' {
				return false
			}
			return true
		})
	}

	// Return
	return keywordType, keywordContents
}

func lexCode(code string) parsedCode {
	text := textAndPosition{
		text:  code,
		index: 0,
		location: textLocation{
			line:   1,
			column: 1,
		},
	}
	var keywords []keyword
	var parsingErrors []codeParsingError
	var nesting uint8

	for text.findUntil(isNotIgnoreableWhitespace) {
		keywordType := UnknownKeywordType
		keywordContents := ""
		keywordPosition := text.location

		switch text.text[text.index] {
		case '\n':
			keywordType = Newline
			keywordContents = "\n"
			if text.moveForward() {
				return parsedCode{
					keywords:      keywords,
					parsingErrors: parsingErrors,
				}
			}
		case '#':
			keywordType = Comment
			keywordContents = text.findUntilWithIteratedString(func(charecter byte) bool {
				if charecter == '\n' {
					return true
				}
				return false
			})

		case '(', '{', '[':
			keywordType = IncreaseNesting
			keywordContents = string(text.text[text.index])
			text.moveForward()
		case ')', '}', ']':
			keywordType = DecreaseNesting
			keywordContents = string(text.text[text.index])
			nesting -= 1
			text.moveForward()

		case '"':
			keywordType = StringValue
			keywordContents = "\""
			text.moveForward()
			keywordContents += text.findUntilWithIteratedString(func(charecter byte) bool {
				if charecter == '"' {
					return true
				}
				return false
			})
			keywordContents += "\""
			text.moveForward()

		case ',', ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%', '!', '^':
			// Get a list of consecutively used syntax symbols
			keywordContents = string(text.text[text.index])
			text.moveForward()
			text.findUntil(func(charecter byte) bool {
				if !isNotIgnoreableWhitespace(charecter) {
					return false
				}
				switch charecter {
				case ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%':
					keywordContents += string(charecter)
					return false
				}
				return true
			})

			// Depending on what syntax symbols were consecutively used, add a keyword or
			// create an error message.
			switch keywordContents {
			case "::":
				keywordType = VariableCreationSyntax
			case "=", "++", "--", "+=", "-=", "*=", "/=", "%=":
				keywordType = VariableModificationSyntax
			case "==", "!=", "<=", ">=", "<", ">":
				keywordType = ComparisonSyntax
			case "^":
				keywordType = PointerOf
			case "-": // The keyword is a negative number
				text.moveForward()
				if text.text[text.index] < '0' || text.text[text.index] > '9' {
					add(&parsingErrors, codeParsingError{
						msg:      errors.New("After -, expecting a number"),
						location: keywordPosition,
					})
					continue
				}
				keywordType, keywordContents = numberToKeyword(&text)
				keywordContents = "-" + keywordContents
			case ",":
				keywordType = ListSyntax
			case "...":
				keywordType = AntiListSyntax
			case "..":
				keywordType = IterationSyntax
			default:
				add(&parsingErrors, codeParsingError{
					msg: errors.New(
						"Unknown symbols series `" +
							keywordContents +
							"`. Known symbol serieses are (, {, [, ), }, ], #, :=, ::, =, |>, ==, ||, &&, <=, >=, <, >, +, -, *, /, %, ,, ..., .., .",
					),
					location: keywordPosition,
				})
				continue
			}

		case '_':
			keywordType = Name
			keywordContents = "_"
			text.moveForward()
		case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
			'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
			keywordContents = text.findUntilWithIteratedString(func(charecter byte) bool {
				if ('a' <= charecter && charecter <= 'z') ||
					('A' <= charecter && charecter <= 'Z') ||
					('0' <= charecter && charecter <= '9') {
					return false
				}
				return true
			})
			switch keywordContents {
			case "if", "else", "while":
				keywordType = ControlFlowSyntax
			case "true", "false":
				keywordType = BoolValue
			case "mut", "arg", "mutArg":
				keywordType = Mutatability
			case "any", "rsi", "rdx", "rax", "rdi", "ecx", "rbx", "bl":
				// TODO: Rethink the register names so that they are not specefic to x86,
				// and are easier to understand for people that come from higher level
				// languages.
				keywordType = Register
			case "syscall":
				keywordType = Syscall
			case "return":
				keywordType = FunctionReturn
			case "import":
				keywordType = Import
			case "and", "or":
				keywordType = ComparisonSyntax
			default:
				keywordType = Name
				text.findUntil(isNotIgnoreableWhitespace)
				if text.text[text.index] == '.' {
					keywordContents += "."
					text.index++
					text.findUntil(isNotIgnoreableWhitespace)
					keywordContents += text.findUntilWithIteratedString(func(charecter byte) bool {
						if ('0' <= charecter && charecter <= '9') || charecter == '_' {
							return false
						}
						return true
					})
				}
			}

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			keywordType, keywordContents = numberToKeyword(&text)

		default:
			add(&parsingErrors, codeParsingError{
				msg: errors.New(
					"Unexpected charecter: `" + string(text.text[text.index]) + "`",
				),
				location: keywordPosition,
			})
			text.moveForward()
			continue
		}

		assert(keywordType != UnknownKeywordType)
		add(&keywords, keyword{
			keywordType: keywordType,
			contents:    keywordContents,
			nesting:     nesting,
			location:    keywordPosition,
		})

		if keywordType == IncreaseNesting {
			// Increasing nesting must go after the keywords append operation so that
			// the ({[ do not get counted as having increased nesting compared to the
			// keyword before them.
			nesting += 1
		}
	}

	return parsedCode{
		keywords:      keywords,
		parsingErrors: parsingErrors,
	}
}
