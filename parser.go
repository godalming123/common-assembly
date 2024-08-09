package main

// Parser.go
// =========
// Responsible for parsing code into a list of keywords and errors
// that are encountered during the process.

import (
	"errors"
	"os"
)

///////////////////////
// TEXT AND POSITION //
///////////////////////

// Stores a position within a piece of text
type textAndPosition struct {
	text   []byte
	index  int
	line   int
	column int
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
		text.column = 0
		text.line += 1
	} else {
		text.column += 1
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

// Same as `findUntil`, but returns the bytes that were iterated over instead
// of true or false.
func (text *textAndPosition) findUntilWithIteratedBytes(checker func(byte) bool) []byte {
	start := text.index
	text.findUntil(checker)
	return text.text[start:text.index]
}

//////////////////
// KEYWORD TYPE //
//////////////////

const (
	Name                       = iota // myVariableName3, _
	StringValue                       // "John"
	BoolValue                         // true, false
	Number                            // 4, 23
	Period                            // . - Can be used to create floats (3.2) or access values in structs (person.name)
	IncreaseNesting                   // (, {, [
	DecreaseNesting                   // ), }, ]
	VariableCreationSyntax            // ::, :=
	VariableModificationSyntax        // =
	ControlFlowSyntax                 // for, if, if, else, |>
	ComparisonSyntax                  // >, <, >=, <=, ||, &&
	ListSyntax                        // ,
	AntiListSyntax                    // ...
	IterationSyntax                   // ..
	MathSyntax                        // +, -, *, /
	Comment                           // # My comment 2
	UnknownKeywordType
)

func convertKeywordTypeToString(keywordType uint8) string {
	// WHY CAN'T GO JUST HAVE ACTUAL ENUM TYPES? THEN THIS WOULD BE AS SIMPLE AS
	// `enumToString(keywordType)`.
	switch keywordType {
	case Name:
		return "Name"
	case StringValue:
		return "StringValue"
	case BoolValue:
		return "BoolValue"
	case Number:
		return "Number"
	case Period:
		return "Period"
	case IncreaseNesting:
		return "IncreaseNesting"
	case DecreaseNesting:
		return "DecreaseNesting"
	case VariableCreationSyntax:
		return "VariableCreationSyntax"
	case VariableModificationSyntax:
		return "VariableModificationSyntax"
	case ControlFlowSyntax:
		return "ControlFlowSyntax"
	case ComparisonSyntax:
		return "ComparisonSyntax"
	case ListSyntax:
		return "ListSyntax"
	case AntiListSyntax:
		return "AntiListSyntax"
	case IterationSyntax:
		return "IterationSyntax"
	case MathSyntax:
		return "MathSyntax"
	case Comment:
		return "Comment"
	default:
		return "UnknownKeywordType"
	}
}

type keyword struct {
	contents    []byte
	keywordType uint8
	line        int
	column      int
}

type keywordList []keyword

func (keywords *keywordList) append(keywordToAppend keyword) {
	*keywords = append(*keywords, keywordToAppend)
}

/////////////////////////////
// CODE PARSING ERROR TYPE //
/////////////////////////////

type codeParsingError struct {
	errorMsg error
	line     int
	column   int
}

type codeParsingErorList []codeParsingError

func (codeParsingErrors *codeParsingErorList) append(codeParsingErrorToAppend codeParsingError) {
	*codeParsingErrors = append(*codeParsingErrors, codeParsingErrorToAppend)
}

//////////////////////
// PARSED CODE TYPE //
//////////////////////

type parsedCode struct {
	keywords      keywordList
	parsingErrors []codeParsingError
}

////////////////////////
// OTHER  HELPER CODE //
////////////////////////

func isNotWhitespace(charecter byte) bool {
	if charecter == ' ' || charecter == '\n' || charecter == '\t' {
		return false
	}
	return true
}

///////////////
// MAIN CODE //
///////////////

func convertFileIntoParsedCode(filePath string) parsedCode {
	rawText, err := os.ReadFile(filePath)
	if err != nil {
		return parsedCode{
			parsingErrors: []codeParsingError{
				{errorMsg: err},
			},
		}
	}

	text := textAndPosition{
		text:   rawText,
		index:  0,
		line:   1,
		column: 0,
	}
	var keywords keywordList
	var parsingErrors codeParsingErorList

	for text.findUntil(isNotWhitespace) {
		var keywordType uint8 = UnknownKeywordType
		keywordContents := []byte{}
		keywordPositionLine := text.line
		keywordPositionColumn := text.column

		switch text.text[text.index] {
		case '#':
			keywordType = Comment
			keywordContents = text.findUntilWithIteratedBytes(func(charecter byte) bool {
				if charecter == '\n' {
					return true
				}
				return false
			})

		case '(', '{', '[':
			keywordType = IncreaseNesting
			keywordContents = []byte{text.text[text.index]}
			text.moveForward()
		case ')', '}', ']':
			keywordType = DecreaseNesting
			keywordContents = []byte{text.text[text.index]}
			text.moveForward()

		case '"':
			keywordType = StringValue
			keywordContents = []byte{'"'}
			text.moveForward()
			keywordContents = append(
				keywordContents,
				text.findUntilWithIteratedBytes(func(charecter byte) bool {
					if charecter == '"' {
						return true
					}
					return false
				})...,
			)
			keywordContents = append(keywordContents, '"')
			text.moveForward()

		case ',', ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%':
			// Get a list of consecutively used syntax symbols
			keywordContents = append(keywordContents, text.text[text.index])
			text.moveForward()
			text.findUntil(func(charecter byte) bool {
				if !isNotWhitespace(charecter) {
					return false
				}
				switch charecter {
				case ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%':
					keywordContents = append(keywordContents, charecter)
					return false
				}
				return true
			})

			// Depending on what syntax symbols were consecutively used, add a keyword or
			// create an error message.
			switch string(keywordContents) {
			case ":=", "::":
				keywordType = VariableCreationSyntax
			case "=":
				keywordType = VariableModificationSyntax
			case "|>":
				keywordType = ControlFlowSyntax
			case "==", "||", "&&", "<=", ">=", "<", ">":
				keywordType = ComparisonSyntax
			case "+", "-", "*", "/", "%":
				keywordType = MathSyntax
			case ",":
				keywordType = ListSyntax
			case "...":
				keywordType = AntiListSyntax
			case "..":
				keywordType = IterationSyntax
			case ".":
				keywordType = Period
			default:
				parsingErrors.append(codeParsingError{
					errorMsg: errors.New(
						"Unknown symbols series `" +
							string(keywordContents) +
							"`. Known symbol serieses are (, {, [, ), }, ], #, :=, ::, =, |>, ==, ||, &&, <=, >=, <, >, +, -, *, /, %, ,, ..., .., .",
					),
					line:   keywordPositionLine,
					column: keywordPositionColumn,
				})
				continue
			}

		case '_':
			keywordType = Name
			keywordContents = []byte{text.text[text.index]}
			text.moveForward()
		case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
			'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
			keywordContents = text.findUntilWithIteratedBytes(func(charecter byte) bool {
				if ('a' <= charecter && charecter <= 'z') ||
					('A' <= charecter && charecter <= 'Z') ||
					('0' <= charecter && charecter <= '9') {
					return false
				}
				return true
			})
			switch string(keywordContents) {
			case "if", "else", "for", "in":
				keywordType = ControlFlowSyntax
			case "true", "false":
				keywordType = BoolValue
			default:
				keywordType = Name
			}

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			keywordType = Number
			keywordContents = text.findUntilWithIteratedBytes(func(charecter byte) bool {
				if ('0' <= charecter && charecter <= '9') || charecter == '_' {
					return false
				}
				return true
			})

		default:
			parsingErrors.append(codeParsingError{
				errorMsg: errors.New(
					"Unexpected charecter: `" + string(text.text[text.index]) + "`",
				),
				line:   keywordPositionLine,
				column: keywordPositionColumn,
			})
			text.moveForward()
			continue
		}

		keywords.append(keyword{
			keywordType: keywordType,
			contents:    keywordContents,
			line:        keywordPositionLine,
			column:      keywordPositionColumn,
		})
	}

	return parsedCode{
		keywords:      keywords,
		parsingErrors: parsingErrors,
	}
}
