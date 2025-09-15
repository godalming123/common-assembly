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

// This type is used to represent the type of keyword that a keyword is. The
// table below shows what kind of values would be in `keyword.contents` for
// a keyword of that `keywordType`.
//
//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=keywordType
type keywordType uint8

const (
	Unknown keywordType = iota
	//                // keyword.contents             //
	// -------------- // ---------------------------- //
	Name              // myFuncName1, myVarName2      //
	RegisterKeyword   // b0, b1, b2..., s0, s1, s2... //
	StringValue       // "Foo", "Bar"                 //
	CharValue         // 'a', '\n'                    //
	BoolValue         // true, false                  //
	PositiveInteger   // 4, 23                        //
	NegativeInteger   // -4, -5                       //
	Decimal           // 2.1, 5.8                     //
	IncreaseNesting   // (, {, [                      //
	DecreaseNesting   // ), }, ]                      //
	Function          // fn                           //
	FunctionReturn    // return                       //
	DropVariable      // drop                         //
	Assignment        // =                            //
	Increment         // ++                           //
	Decrement         // --                           //
	PlusEquals        // +=                           //
	MinusEquals       // -=                           //
	MultiplyEquals    // *=                           //
	DivideEquals      // /=                           //
	WhileLoop         // while                        //
	BreakStatement    // break                        //
	ContinueStatement // continue                     //
	IfStatement       // if                           //
	ElifStatement     // elif                         //
	ElseStatement     // else                         //
	ComparisonSyntax  // ==, !=, >, <, >=, <=         //
	And               // and                          //
	Or                // or                           //
	ListSyntax        // ,                            //
	Import            // import                       //
	Dereference       // ^                            //
	Comment           // # My comment 2               //
	Newline           // \n                           //
)

// Stores an individual keyword. When there is a list of keywords, the
// concatenation of all of the keywords contents should be equal to the original
// code that was lexed into the list of keywords, but without `\r`, `\t`, or
// spaces that aren't a part of a StringValue or CharValue keyword.
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
	for i := range keywords {
		keywords[i].contents = strings.Replace(keywords[i].contents, "\n", "\\n", -1)
		keywords[i].contents = strings.Replace(keywords[i].contents, "\t", "    ", -1)
		if len(fmt.Sprint(keywords[i].location.line)) > longestLine {
			longestLine = len(fmt.Sprint(keywords[i].location.line))
		}
		if len(fmt.Sprint(keywords[i].location.column)) > longestColumn {
			longestColumn = len(fmt.Sprint(keywords[i].location.column))
		}
		if len(fmt.Sprint(int(keywords[i].nesting))) > longestNesting {
			longestNesting = len(fmt.Sprint(keywords[i].nesting))
		}
		if len(keywords[i].keywordType.String()) > longestType {
			longestType = len(keywords[i].keywordType.String())
		}
		if len(keywords[i].contents) > longestContents {
			longestContents = len(strings.Replace(keywords[i].contents, "\n", "\\n", -1))
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
			{contents: keyword.contents, width: longestContents},
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
	msg error
	textLocation
}

////////////////////////
// OTHER  HELPER CODE //
////////////////////////

func isNotIgnorableWhitespace(character byte) bool {
	if character == ' ' || character == '\t' || character == '\r' {
		return false
	}
	return true
}

func isNotNumber(character byte) bool {
	if ('0' <= character && character <= '9') || character == '_' {
		return false
	}
	return true
}

func isNotVariableCharacter(character byte) bool {
	if ('a' <= character && character <= 'z') ||
		('A' <= character && character <= 'Z') ||
		('0' <= character && character <= '9') ||
		character == '_' {
		return false
	}
	return true
}

///////////////
// MAIN CODE //
///////////////

// The returned bool is true if the number is a decimal, and false otherwise
func positiveNumberToKeyword(text *textAndPosition) (bool, string) {
	// Parse any digits (and `_`) into keywordContents
	keywordContents := text.findUntilWithIteratedString(isNotNumber)

	// Early return if their is not a decimal after the number
	if text.index >= len(text.text)-1 ||
		text.text[text.index] != '.' ||
		text.text[text.index+1] == '.' {
		return false, keywordContents
	}

	// Handle the decimal
	keywordContents += "."
	assert(eq(text.moveForward(), false))
	return true, keywordContents + text.findUntilWithIteratedString(isNotNumber)
}

func lexCode(code string) ([]keyword, []codeParsingError) {
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

	for text.findUntil(isNotIgnorableWhitespace) {
		keywordType := Unknown
		keywordContents := ""
		keywordPosition := text.location

		switch text.text[text.index] {
		case '\n':
			keywordType = Newline
			keywordContents = "\n"
			if text.moveForward() {
				return keywords, parsingErrors
			}
		case '#':
			keywordType = Comment
			keywordContents = text.findUntilWithIteratedString(func(character byte) bool {
				if character == '\n' {
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

		case '\'':
			keywordType = CharValue
			keywordContents = "'"
			if text.moveForward() {
				add(&parsingErrors, codeParsingError{
					msg:          errors.New("Unexpected end of text while parsing character value"),
					textLocation: text.location,
				})
			}
			if text.text[text.index] == '\\' {
				keywordContents += "\\"
				if text.moveForward() {
					add(&parsingErrors, codeParsingError{
						msg:          errors.New("Unexpected end of text while parsing character value"),
						textLocation: text.location,
					})
				}
			}
			keywordContents += string(text.text[text.index]) + "'"
			if text.moveForward() {
				add(&parsingErrors, codeParsingError{
					msg:          errors.New("Unexpected end of text while parsing character value"),
					textLocation: text.location,
				})
			}
			if text.text[text.index] != '\'' {
				add(&parsingErrors, codeParsingError{
					msg:          errors.New("Expected `'' to end character value, got `" + string(text.text[text.index]) + "`"),
					textLocation: text.location,
				})
			}
			text.moveForward()

		case '"':
			keywordType = StringValue
			keywordContents = "\""
			text.moveForward()
			keywordContents += text.findUntilWithIteratedString(func(character byte) bool {
				if character == '"' {
					return true
				}
				return false
			})
			keywordContents += "\""
			text.moveForward()

		case ',', ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%', '!', '^':
			// Get a list of consecutively used syntax symbols. We cannot use
			// `findUntilWithIteratedString` since that would add the ignorable
			// whitespace to the string.
			keywordContents = string(text.text[text.index])
			text.moveForward()
			text.findUntil(func(character byte) bool {
				if !isNotIgnorableWhitespace(character) {
					return false
				}
				switch character {
				case ':', '=', '|', '<', '>', '&', '+', '-', '*', '/', '.', '%':
					keywordContents += string(character)
					return false
				}
				return true
			})

			// Depending on what syntax symbols were consecutively used, add a keyword or
			// create an error message.
			switch keywordContents {
			case "=":
				keywordType = Assignment
			case "++":
				keywordType = Increment
			case "--":
				keywordType = Decrement
			case "+=":
				keywordType = PlusEquals
			case "-=":
				keywordType = MinusEquals
			case "*=":
				keywordType = MultiplyEquals
			case "/=":
				keywordType = DivideEquals
			case "==", "!=", "<=", ">=", "<", ">":
				keywordType = ComparisonSyntax
			case "^":
				keywordType = Dereference
			case "-": // The keyword is a negative number
				text.moveForward()
				if text.text[text.index] < '0' || text.text[text.index] > '9' {
					add(&parsingErrors, codeParsingError{
						msg:          errors.New("After `-`, expecting a number"),
						textLocation: keywordPosition,
					})
					continue
				}
				hasDecimal := false
				hasDecimal, keywordContents = positiveNumberToKeyword(&text)
				keywordContents = "-" + keywordContents
				if hasDecimal {
					keywordType = Decimal
				} else {
					keywordType = NegativeInteger
				}
			case ",":
				keywordType = ListSyntax
			default:
				add(&parsingErrors, codeParsingError{
					msg: errors.New(
						"Unknown symbols series `" +
							keywordContents +
							"`. Known symbol series are (, {, [, ), }, ], #, :=, ::, =, |>, ==, ||, &&, <=, >=, <, >, +, -, *, /, %, ,, ..., .., .",
					),
					textLocation: keywordPosition,
				})
				continue
			}

		case '_', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
			'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
			keywordContents = text.findUntilWithIteratedString(isNotVariableCharacter)
			switch keywordContents {
			case "fn":
				keywordType = Function
			case "drop":
				keywordType = DropVariable
			case "if":
				keywordType = IfStatement
			case "elif":
				keywordType = ElifStatement
			case "else":
				keywordType = ElseStatement
			case "while":
				keywordType = WhileLoop
			case "break":
				keywordType = BreakStatement
			case "continue":
				keywordType = ContinueStatement
			case "true", "false":
				keywordType = BoolValue
			case "r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7",
				"r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15":
				keywordType = RegisterKeyword
			case "return":
				keywordType = FunctionReturn
			case "import":
				keywordType = Import
			case "and":
				keywordType = And
			case "or":
				keywordType = Or
			default:
				keywordType = Name
				text.findUntil(isNotIgnorableWhitespace)
				if text.text[text.index] == '.' {
					keywordContents += "."
					text.index++
					text.findUntil(isNotIgnorableWhitespace)
					keywordContents += text.findUntilWithIteratedString(func(character byte) bool {
						if ('0' <= character && character <= '9') || character == '_' {
							return false
						}
						return true
					})
				}
			}

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			hasDecimal := false
			hasDecimal, keywordContents = positiveNumberToKeyword(&text)
			if hasDecimal {
				keywordType = Decimal
			} else {
				keywordType = PositiveInteger
			}

		default:
			add(&parsingErrors, codeParsingError{
				msg: errors.New(
					"Unexpected character: `" + string(text.text[text.index]) + "`",
				),
				textLocation: keywordPosition,
			})
			text.moveForward()
			continue
		}

		assert(notEq(keywordType, Unknown))
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

	return keywords, parsingErrors
}
