package main

// Parser.go
// =========
// Responsible for parsing common assembly into a list of keywords and errors
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
	text   string
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

type keywordType uint8

const (
	Name                       keywordType = iota // myVariableName3, _
	StringValue                                   // "John"
	BoolValue                                     // true, false
	Number                                        // 4, 23
	Period                                        // . - Can be used to create floats (3.2) or access values in structs (person.name)
	IncreaseNesting                               // (, {, [
	DecreaseNesting                               // ), }, ]
	VariableCreationSyntax                        // ::, :=
	VariableModificationSyntax                    // =
	ControlFlowSyntax                             // while, if, else
	ComparisonSyntax                              // >, <, >=, <=, ||, &&
	ListSyntax                                    // ,
	AntiListSyntax                                // ...
	IterationSyntax                               // ..
	MathSyntax                                    // +, -, *, /
	Comment                                       // # My comment 2
	BuiltInFunction                               // syscall, import
	Newline                                       // \n
	UnknownKeywordType
)

func convertKeywordTypeToString(typeToConvert keywordType) string {
	// WHY CAN'T GO JUST HAVE ACTUAL ENUM TYPES? THEN THIS WOULD BE AS SIMPLE AS
	// `enumToString(keywordType)`.
	switch typeToConvert {
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
	case BuiltInFunction:
		return "BuiltInFuction"
	default:
		return "UnknownKeywordType"
	}
}

type keyword struct {
	contents    string
	keywordType keywordType
	nesting     uint8
	line        int
	column      int
}

/////////////////////////////
// CODE PARSING ERROR TYPE //
/////////////////////////////

type codeParsingError struct {
	msg    error
	line   int
	column int
	level  logLevel
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

func convertFileIntoParsedCode(filePath string) parsedCode {
	rawText, err := os.ReadFile(filePath)
	if err != nil {
		return parsedCode{
			parsingErrors: []codeParsingError{
				{msg: err},
			},
		}
	}

	text := textAndPosition{
		text:   string(rawText),
		index:  0,
		line:   1,
		column: 0,
	}
	var keywords []keyword
	var parsingErrors []codeParsingError
	var nesting uint8

	for text.findUntil(isNotIgnoreableWhitespace) {
		keywordType := UnknownKeywordType
		keywordContents := ""
		keywordPositionLine := text.line
		keywordPositionColumn := text.column

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

		case ',', ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%':
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
			case ":=", "::":
				keywordType = VariableCreationSyntax
			case "=":
				keywordType = VariableModificationSyntax
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
				add(&parsingErrors, codeParsingError{
					msg: errors.New(
						"Unknown symbols series `" +
							keywordContents +
							"`. Known symbol serieses are (, {, [, ), }, ], #, :=, ::, =, |>, ==, ||, &&, <=, >=, <, >, +, -, *, /, %, ,, ..., .., .",
					),
					line:   keywordPositionLine,
					column: keywordPositionColumn,
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
			case "syscall", "import":
				keywordType = BuiltInFunction
			default:
				keywordType = Name
			}

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			keywordType = Number
			keywordContents = text.findUntilWithIteratedString(func(charecter byte) bool {
				if ('0' <= charecter && charecter <= '9') || charecter == '_' {
					return false
				}
				return true
			})

		default:
			add(&parsingErrors, codeParsingError{
				msg: errors.New(
					"Unexpected charecter: `" + string(text.text[text.index]) + "`",
				),
				line:   keywordPositionLine,
				column: keywordPositionColumn,
			})
			text.moveForward()
			continue
		}

		add(&keywords, keyword{
			keywordType: keywordType,
			contents:    keywordContents,
			nesting:     nesting,
			line:        keywordPositionLine,
			column:      keywordPositionColumn,
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
