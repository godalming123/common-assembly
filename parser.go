package main

import (
	"errors"
	"fmt"
	"strings"
)

// Parser.go
// =========
// Responsible for parsing a list of keywords that make up a common assembly file into a list of the
// `ASTitem` type below while reporting any syntax errors in the list of keywords that were not
// detected by `convertFileIntoParsedCode`.

////////////////////////////////////
// ABSTRACT SYNTAX TREE ITEM TYPE //
////////////////////////////////////

type ASTitem struct {
	itemType typeOfParsedKeywordOrASTitem
	name     string
	contents []ASTitem
	location textLocation
}

func (AST ASTitem) print(indentation int) {
	print(addWhitespaceToStart(fmt.Sprint(AST.location.line), 7))
	print(" ")
	print(strings.Repeat("    ", indentation))
	print(AST.itemType.String())
	if AST.name != "" {
		print(": `" + AST.name + "`")
	}
	println()
	for _, ASTitem := range AST.contents {
		ASTitem.print(indentation + 1)
	}
}

//////////////////////////////////////////////////
// FUNCTIONS TO CONVERT KEYWORDS INTO []ASTitem //
//////////////////////////////////////////////////

func nextNonEmpty(keywords *listIterator[keyword], errorMsg string) codeParsingError {
	for true {
		if !keywords.next() {
			return codeParsingError{
				msg:      errors.New(errorMsg),
				location: keywords.get().location,
			}
		}
		if keywords.get().keywordType != Newline &&
			keywords.get().keywordType != Comment {
			return codeParsingError{}
		}
	}
	panic("Unreachable")
}

func splitCondition(keywords []keyword, splitType typeOfParsedKeywordOrASTitem) ([][]keyword, codeParsingError) {
	lastSplitIndex := -1
	nesting := 0
	returnValue := [][]keyword{}
	for index, item := range keywords {
		switch item.keywordType {
		case IncreaseNesting:
			if item.contents != "(" {
				return [][]keyword{}, codeParsingError{
					msg:      errors.New("In conditions, keyword of type IncreaseNesting must have contents (, got a keyword with contents " + item.contents),
					location: item.location,
				}
			}
			nesting++
		case DecreaseNesting:
			if item.contents != ")" {
				return [][]keyword{}, codeParsingError{
					msg:      errors.New("In conditions, keyword of type DecreaseNesting must have contents ), got a keyword with contents " + item.contents),
					location: item.location,
				}
			}
			nesting--
			if nesting < 0 {
				return [][]keyword{}, codeParsingError{
					msg:      errors.New("Unexpected ), without a ( before it to match"),
					location: item.location,
				}
			}
		case splitType:
			if nesting > 0 {
				continue
			}
			if index == 0 || index == len(keywords)-1 {
				return [][]keyword{}, codeParsingError{
					msg:      errors.New("Cannot have And or Or at the start or end of a condition"),
					location: item.location,
				}
			}
			add(&returnValue, keywords[lastSplitIndex+1:index])
			lastSplitIndex = index
		}
	}
	if nesting > 0 {
		return [][]keyword{}, codeParsingError{
			msg:      errors.New("In condition, there is " + fmt.Sprint(nesting) + " unclosed ("),
			location: keywords[0].location,
		}
	}
	add(&returnValue, keywords[lastSplitIndex+1:])
	return returnValue, codeParsingError{}
}

func conditionToAST(keywords []keyword) (ASTitem, codeParsingError) {
	// Handle outside brackets
	if keywords[0].contents == "(" && keywords[len(keywords)-1].contents == ")" {
		return conditionToAST(keywords[1 : len(keywords)-1])
	}

	// Handle conditions with `and` in them
	andClauses, err := splitCondition(keywords, And)
	if err.msg != nil {
		return ASTitem{}, err
	}
	if len(andClauses) > 1 {
		return conditionClausesToAST(andClauses, And)
	}

	// Handle conditions with `or` in them
	orClauses, err := splitCondition(keywords, Or)
	if err.msg != nil {
		return ASTitem{}, err
	}
	if len(orClauses) > 1 {
		return conditionClausesToAST(orClauses, Or)
	}

	// Handle if there is only 1 keyword
	if len(keywords) == 1 {
		if keywords[0].keywordType != BoolValue {
			return ASTitem{}, codeParsingError{
				msg:      errors.New("Expect conditions that only have one keyword to be of type BoolValue, got a keyword of type `" + keywords[0].keywordType.String() + "`"),
				location: keywords[0].location,
			}
		}
		assert(or(eq(keywords[0].contents, "true"), eq(keywords[0].contents, "false")))
		return ASTitem{
			name:     keywords[0].contents,
			itemType: BoolValue,
			location: keywords[0].location,
		}, codeParsingError{}
	}

	// Handle comparisons without `and` or `or` in them
	return comparisonToAST(keywords)
}

// Parses function arguments. This logic is also used to parse the values in return statements.
// After a succesful execution of this function, `keywords.get()` returns the `)` for functions or
// `}` for returns at the end of the arguments.
func parseFunctionArguments(keywords *listIterator[keyword]) ([]ASTitem, codeParsingError) {
	functionArguments := []ASTitem{}
	if keywords.get().keywordType == DecreaseNesting {
		return []ASTitem{}, codeParsingError{}
	}
	for true {
		register := ""
		if keywords.get().keywordType == Register {
			// Parse register
			register = keywords.get().contents
			err := nextNonEmpty(keywords, "After function argument register, unexpected end of keywords")
			if err.msg != nil {
				return []ASTitem{}, err
			}

			// Parse =
			if keywords.get().keywordType != Assignment || keywords.get().contents != "=" {
				return []ASTitem{}, codeParsingError{
					location: keywords.get().location,
					msg:      errors.New("Expected keyword of type VariableMutation with contents =, got `" + keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
				}
			}
			err = nextNonEmpty(keywords, "While parsing function arguments, after =, unexpected end of keywords")
			if err.msg != nil {
				return []ASTitem{}, err
			}
		}

		// Parse value
		valueAST, err := parseValue(keywords)
		if err.msg != nil {
			return []ASTitem{}, err
		}
		err = nextNonEmpty(keywords, "After value, unexpected end of keywords")
		if err.msg != nil {
			return []ASTitem{}, err
		}

		// Append to arguments
		add(&functionArguments, ASTitem{
			itemType: Register,
			location: valueAST.location,
			name:     register,
			contents: []ASTitem{valueAST},
		})

		// Parse ,
		if keywords.get().keywordType != ListSyntax {
			return functionArguments, codeParsingError{}
		}
		err = nextNonEmpty(keywords, "After `,`, unexpected end of keywords")
		if err.msg != nil {
			return []ASTitem{}, err
		}
	}
	panic("Unreachable")
}

// Parses values including variable names, function calls, numbers, registers, strings, charecters,
// dropped variables, and pointers of things. After a succsesful execution of this function,
// keywords.get() should return the keyword at the end of the value.
func parseValue(keywords *listIterator[keyword]) (ASTitem, codeParsingError) {
	switch keywords.get().keywordType {
	case Name:
		// Parse name
		name := keywords.get()
		oldKeywordsIndex := keywords.currentIndex
		err := nextNonEmpty(keywords, "After keyword of type Name, unexpected end of keywords")

		// Early return if this is a variable reference, and not a funcction call
		if err.msg != nil || keywords.get().keywordType != IncreaseNesting || keywords.get().contents != "(" {
			keywords.currentIndex = oldKeywordsIndex
			return ASTitem{
				itemType: Name,
				name:     name.contents,
				location: name.location,
			}, codeParsingError{}
		}

		// Parse (
		err = nextNonEmpty(keywords, "After Name and then (, unexpected end of keywords")
		if err.msg != nil {
			return ASTitem{}, err
		}

		// Parse arguments
		functionArguments, err := parseFunctionArguments(keywords)

		// Parse )
		if keywords.get().keywordType != DecreaseNesting || keywords.get().contents != ")" {
			return ASTitem{}, codeParsingError{
				location: keywords.get().location,
				msg:      errors.New("Expected keyword of type DecreaseNesting with contents `)`, got `" + keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
			}
		}

		// Return
		return ASTitem{
			itemType: Function,
			name:     name.contents,
			location: name.location,
			contents: functionArguments,
		}, codeParsingError{}
	case IntNumber, FloatNumber, Register:
		return ASTitem{
			name:     keywords.get().contents,
			itemType: keywords.get().keywordType,
			location: keywords.get().location,
		}, codeParsingError{}
	case StringValue, CharValue:
		if keywords.get().keywordType == StringValue {
			assert(eq(keywords.get().contents[0], '"'))
			assert(eq(keywords.get().contents[len(keywords.get().contents)-1], '"'))
		} else {
			assert(eq(keywords.get().contents[0], '\''))
			assert(eq(keywords.get().contents[len(keywords.get().contents)-1], '\''))
		}
		return ASTitem{
			name:     keywords.get().contents[1 : len(keywords.get().contents)-1],
			itemType: keywords.get().keywordType,
			location: keywords.get().location,
		}, codeParsingError{}
	case DropVariable:
		location := keywords.get().location
		err := nextNonEmpty(keywords, "After a keyword of type DropVariable, unexpected end of keywords")
		if err.msg != nil {
			return ASTitem{}, err
		}
		if keywords.get().keywordType != Name {
			return ASTitem{}, codeParsingError{
				msg:      errors.New("After a keyword of type DropVariable, expect a keyword of type Name, got a keyword of type " + keywords.get().keywordType.String()),
				location: keywords.get().location,
			}
		}
		return ASTitem{
			itemType: DropVariable,
			name:     keywords.get().contents,
			location: location,
		}, codeParsingError{}
	case Dereference:
		keyword := keywords.get()
		if !keywords.next() {
			return ASTitem{}, codeParsingError{
				msg:      errors.New("Unexepected end of keywords"),
				location: keywords.get().location,
			}
		}
		subAST, err := parseValue(keywords)
		if err.msg != nil {
			return ASTitem{}, err
		}
		if subAST.itemType != DropVariable &&
			subAST.itemType != Dereference &&
			subAST.itemType != Name {
			return ASTitem{}, codeParsingError{
				msg:      errors.New("Expect ^ to point to a value of type DropVariable, Dereference or Name, got a value of type " + subAST.itemType.String()),
				location: keywords.get().location,
			}
		}
		return ASTitem{
			itemType: Dereference,
			contents: []ASTitem{subAST},
			location: keyword.location,
		}, codeParsingError{}
	default:
		return ASTitem{}, codeParsingError{
			msg:      errors.New("While parsing value, unexpected keyword type " + keywords.get().keywordType.String() + " expecting a keyword of type Name, IntNumber, FloatNumber, StringValue, CharValue, Dereference, or DropVariable"),
			location: keywords.get().location,
		}
	}
}

// Parses a comparison into an AST node. `keywords` cannot contain a keyword
// where `contents == "and"` or `contents == "or"`, or else this function will
// panic. If `keywords` may contain a keyword where
// `contents == "and"` or `contents == "or"`, then use `conditionToAST`.
func comparisonToAST(keywordList []keyword) (ASTitem, codeParsingError) {
	assert(greaterThan(len(keywordList), 0))

	var comparisonType byte
	comparisonClauses := []ASTitem{}
	keywords := listIterator[keyword]{list: keywordList}

	for true {
		comparisonFirstArg, err := parseValue(&keywords)
		if err.msg != nil {
			return ASTitem{}, err
		}
		if nextNonEmpty(&keywords, "Unexpected end of keywords").msg != nil {
			if keywords.currentIndex == 0 {
				return ASTitem{}, codeParsingError{
					location: keywords.get().location,
					msg:      errors.New("Unexpected end of comparison, expecting either >, >=, <, <=, ==, or !="),
				}
			} else if len(comparisonClauses) == 1 {
				return comparisonClauses[0], codeParsingError{}
			} else {
				return ASTitem{
					itemType: And,
					contents: comparisonClauses,
					location: keywordList[0].location,
				}, codeParsingError{}
			}
		}

		comparisonKeyword := keywords.get()
		if comparisonKeyword.keywordType != ComparisonSyntax {
			return ASTitem{}, codeParsingError{
				location: comparisonKeyword.location,
				msg:      errors.New("Expecting a keyword of type ComparisonSyntax (==, !=, >, <, >=, <=), got a keyword of type " + comparisonKeyword.keywordType.String() + "."),
			}
		}
		if comparisonType == 0 {
			// If this is the first iteration of the loop, then set comparisonType
			comparisonType = comparisonKeyword.contents[0]
			if comparisonType != '=' && comparisonType != '!' &&
				comparisonType != '<' && comparisonType != '>' {
				panic("Unexpected internal state: `comparisonToAST` got a keyword with " +
					"contents `" +
					comparisonKeyword.contents +
					"` expecting the keyword contents to start with either =, !, <, >.")
			}
		} else {
			// If comparisonType is already set to a non-zero value, then this is past the first itereation
			// of the for loop, and we should check that the comparison is valid given comparisonType
			if comparisonType != comparisonKeyword.contents[0] {
				return ASTitem{}, codeParsingError{
					location: comparisonKeyword.location,
					msg:      errors.New("Expecting comparisons in greatness chain to match"),
				}
			}
			if comparisonType == '!' {
				return ASTitem{}, codeParsingError{
					location: comparisonKeyword.location,
					msg:      errors.New("You cannot chain comparisons of type !"),
				}
			}
		}

		err = nextNonEmpty(&keywords, "During parsing of comparison, after `"+keywords.get().contents+
			"`, unexpected end of keywords. Expected a value.")
		if err.msg != nil {
			return ASTitem{}, err
		}

		comparisonSecondArg, err := parseValue(&keywords)
		if err.msg != nil {
			return ASTitem{}, err
		}

		add(&comparisonClauses, ASTitem{
			itemType: ComparisonSyntax,
			name:     comparisonKeyword.contents,
			location: comparisonKeyword.location,
			contents: []ASTitem{
				comparisonFirstArg,
				comparisonSecondArg,
			},
		})
	}
	panic("Unreachable")
}

func conditionClausesToAST(clauses [][]keyword, conditionFunctionType typeOfParsedKeywordOrASTitem) (ASTitem, codeParsingError) {
	conditionClausesAST := []ASTitem{}
	for _, clause := range clauses {
		clauseASTitem, err := conditionToAST(clause)
		if err.msg != nil {
			return ASTitem{}, err
		}
		add(&conditionClausesAST, clauseASTitem)
	}
	return ASTitem{
		location: clauses[0][0].location,
		itemType: conditionFunctionType,
		contents: conditionClausesAST,
	}, codeParsingError{}
}

// Parses a "conditional block" into an AbstractSyntaxTree node. This consists
// of ignoring the first keyword, then parsing a condition, then parsing a
// block, and then concatonating the result into one AST item of type
// `ASTitemType`.
func conditionalBlockToAST(
	keywords *listIterator[keyword],
	ASTitemType typeOfParsedKeywordOrASTitem,
) (ASTitem, codeParsingError) {
	// Save the location
	location := keywords.get().location

	// Ignore the first keyword
	if !keywords.next() {
		return ASTitem{}, codeParsingError{
			msg:      errors.New("During parsing of the conditonal block, unexpected end of keywords slice."),
			location: keywords.get().location,
		}
	}

	// Get the keywords in the condition
	conditionKeywords := []keyword{}
	for keywords.get().contents != "{" {
		add(&conditionKeywords, *keywords.get())
		if !keywords.next() {
			return ASTitem{}, codeParsingError{
				msg:      errors.New("Unexpected end of keywords."),
				location: keywords.get().location,
			}
		}
	}

	// Parse the condition into AST
	conditionAST, err := conditionToAST(conditionKeywords)
	if err.msg != nil {
		return ASTitem{}, err
	}

	// Parse the block into AST
	blockAST, err := blockToAST(keywords)
	if err.msg != nil {
		return ASTitem{}, err
	}

	// Return
	return ASTitem{
		itemType: ASTitemType,
		contents: append([]ASTitem{conditionAST}, blockAST...),
		location: location,
	}, codeParsingError{}
}

func ifStatementToAST(keywords *listIterator[keyword]) (ASTitem, codeParsingError) {
	// Parse if block
	out, err := conditionalBlockToAST(keywords, IfStatement)
	if err.msg != nil {
		return ASTitem{}, err
	}

	// Parse else block if there is one
	if keywords.currentIndex+1 < len(keywords.list) {
		if keywords.list[keywords.currentIndex+1].contents == "elif" {
			assert(eq(keywords.next(), true))
			elifBlock, err := ifStatementToAST(keywords)
			if err.msg != nil {
				return ASTitem{}, err
			}
			add(&out.contents, ASTitem{
				location: elifBlock.location,
				itemType: ElseStatement,
				contents: []ASTitem{elifBlock},
			})
		} else if keywords.list[keywords.currentIndex+1].contents == "else" {
			assert(eq(keywords.next(), true))
			if !keywords.next() {
				return ASTitem{}, codeParsingError{
					msg:      errors.New("Unexpected end of block. Either remove the else, or add a block after the else."),
					location: keywords.get().location,
				}
			}
			elseBlockContents, err := blockToAST(keywords)
			if err.msg != nil {
				return ASTitem{}, err
			}
			add(&out.contents, ASTitem{
				location: elseBlockContents[0].location,
				itemType: ElseStatement,
				contents: elseBlockContents,
			})
		}
	}

	// Return
	return out, codeParsingError{}
}

// After a succsesful execution of this function, keywords.get().contents should equal to "}"
func blockToAST(keywords *listIterator[keyword]) ([]ASTitem, codeParsingError) {
	// Parse {
	if keywords.get().contents != "{" {
		return []ASTitem{}, codeParsingError{
			msg:      errors.New("Expecting { to start a new block."),
			location: keywords.get().location,
		}
	}

	ASTitems := []ASTitem{}

	// Parse each statement inside the block
	for true {
		err := nextNonEmpty(keywords, "During the parsing of a block, unexpected end of the keywords slice")
		if err.msg != nil {
			return []ASTitem{}, err
		}
		switch keywords.get().keywordType {
		case FunctionReturn:
			// Save the location of the return
			location := keywords.get().location

			// Move past the return keyword
			err := nextNonEmpty(keywords, "Unexpected end of keywords")
			if err.msg != nil {
				return []ASTitem{}, err
			}

			// Parse the return values
			returnValues, err := parseFunctionArguments(keywords)
			if err.msg != nil {
				return []ASTitem{}, err
			}

			// Parse }
			if keywords.get().keywordType != DecreaseNesting || keywords.get().contents != "}" {
				return []ASTitem{}, codeParsingError{
					location: keywords.get().location,
					msg:      errors.New("Expected keyword of type DecreaseNesting with contents `)`, got `" + keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
				}
			}

			// Return the function
			return append(ASTitems, ASTitem{
				location: location,
				itemType: FunctionReturn,
				contents: returnValues,
			}), codeParsingError{}

		case Function:
			functionDefinitionAST, err := functionDefinitionToAST(keywords)
			if err.msg != nil {
				return []ASTitem{}, err
			}
			add(&ASTitems, functionDefinitionAST)

		case Register, Dereference, DropVariable, Name:
			variableMutationAST, err := variableMutationToAST(keywords)
			if err.msg != nil {
				return []ASTitem{}, err
			}
			add(&ASTitems, variableMutationAST)

		// Statements that start with control flow syntax can either be a while loop
		// or an `if`, `elif`, `else` statement.
		case IfStatement:
			conditionalBlock, err := ifStatementToAST(keywords)
			if err.msg != nil {
				return []ASTitem{}, err
			}
			add(&ASTitems, conditionalBlock)
		case WhileLoop:
			conditionalBlock, err := conditionalBlockToAST(keywords, WhileLoop)
			if err.msg != nil {
				return []ASTitem{}, err
			}
			add(&ASTitems, conditionalBlock)
		case BreakStatement:
			add(&ASTitems, ASTitem{location: keywords.get().location, itemType: BreakStatement})
		case ContinueStatement:
			add(&ASTitems, ASTitem{location: keywords.get().location, itemType: ContinueStatement})

		// The only valid statement that starts with decrease nesting is }, which exits the block scope
		case DecreaseNesting:
			switch keywords.get().contents {
			case "}":
				return ASTitems, codeParsingError{}
			default:
				return []ASTitem{}, codeParsingError{
					msg:      errors.New("Expecting a keyword of type `DecreaseNesting` within a block to have contents `}` got `" + keywords.get().contents + "`."),
					location: keywords.get().location,
				}
			}
		default:
			return []ASTitem{}, codeParsingError{
				msg:      errors.New("Expecting a keyword of type Newline, Comment, Name, ControlFlowSyntax, Register, or DecreaseNesting, got a keyword of type " + keywords.get().keywordType.String()),
				location: keywords.get().location,
			}
		}
	}
	panic("Unreachable")
}

// Parses a statement starting with a keyword of type Name, Register, DropVariable, or Dereference
// into an AST item. Examples of this type of statement include:
// - `b0 returnStutus, b1 = myFunction(b1="test", b2=myVariable)`
// - `drop returnStatus`
// - `b0 result, b1 = power(b1=base, b2=power)`
// - `^pointsToChar = 'a'`
// After a succsesful execution of this function, keywords.get() should return the end of the
// statement, and the returned `ASTitem` could is itemType VariableMutation.
func variableMutationToAST(keywords *listIterator[keyword]) (ASTitem, codeParsingError) {
	// Parse the things that are being mutated
	variableBeingMutated := ASTitem{itemType: SetToAValue, location: keywords.get().location}
	for true {
		// Parse value
		value, err := parseValue(keywords)
		if err.msg != nil {
			return ASTitem{}, err
		}
		if !keywords.next() {
			return ASTitem{}, codeParsingError{
				msg:      errors.New("After value, unexpected end of keywords"),
				location: keywords.get().location,
			}
		}
		switch value.itemType {
		case Name, DropVariable, Dereference:
			value = ASTitem{
				itemType: Register,
				location: value.location,
				contents: []ASTitem{value},
			}
		case Register:
			// There could be a variable name after the register
			if keywords.get().keywordType == Name {
				assert(eq(len(value.contents), 0))
				value.contents = []ASTitem{{
					itemType: Name,
					name:     keywords.get().contents,
					location: keywords.get().location,
				}}
				keywords.next()
			}
		default:
			return ASTitem{}, codeParsingError{
				msg:      errors.New("Expecting value of type Name, Register, DropVariable, or Dereference before =, got " + value.itemType.String()),
				location: value.location,
			}
		}
		add(&variableBeingMutated.contents, value)

		// Parse ,
		if keywords.get().keywordType != ListSyntax {
			break
		}
		err = nextNonEmpty(keywords, "Unexpected end of keywords after ,")
		if err.msg != nil {
			return ASTitem{}, err
		}
	}

	// Parse =, ++, --, +=, -=, *=, /=, and (if needed) `valueBeingAssignedToVariable`
	mutationType := keywords.get().keywordType
	switch mutationType {
	case Increment, Decrement:
		return ASTitem{
			itemType: mutationType,
			location: variableBeingMutated.location,
			contents: []ASTitem{variableBeingMutated},
		}, codeParsingError{}
	case Assignment, PlusEquals, MinusEquals, MultiplyEquals, DivideEquals:
		// Next keyword
		err := nextNonEmpty(keywords, "After `"+keywords.get().contents+"` (variable mutation operator), unexpected end of keywords")
		if err.msg != nil {
			return ASTitem{}, err
		}

		// Parse the value being assigned to the variable
		valueBeingAssignedToVariable, err := parseValue(keywords)
		if err.msg != nil {
			return ASTitem{}, err
		}

		// Return
		return ASTitem{
			location: variableBeingMutated.location,
			itemType: mutationType,
			contents: []ASTitem{variableBeingMutated, valueBeingAssignedToVariable},
		}, codeParsingError{}
	case Comment, Newline:
		if len(variableBeingMutated.contents) == 1 && variableBeingMutated.contents[0].name == "" &&
			variableBeingMutated.contents[0].contents[0].itemType == DropVariable {
			assert(notEq(variableBeingMutated.contents[0].contents[0].name, ""))
			return ASTitem{
				location: variableBeingMutated.contents[0].contents[0].location,
				itemType: DropVariable,
				name:     variableBeingMutated.contents[0].contents[0].name,
			}, codeParsingError{}
		}
		return ASTitem{}, codeParsingError{
			location: keywords.get().location,
			msg: errors.New("After a variable that is being mutated, and not being dropped, expected a " +
				"keyword of type Assignment, Increment, Decrement, PlusEquals, MinusEquals, MultiplyEquals, " +
				"or DivideEquals, got `" + keywords.get().contents + "` of type " +
				keywords.get().keywordType.String()),
		}
	default:
		return ASTitem{}, codeParsingError{
			location: keywords.get().location,
			msg: errors.New("Expected keyword of type Assignment, Increment, Decrement, PlusEquals, " +
				"MinusEquals, MultiplyEquals, DivideEquals, Comment, or Newline, got `" +
				keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
		}
	}
}

// After a succsesful execution of this function, keywords.get().contents should equal to "}"
func functionDefinitionToAST(keywords *listIterator[keyword]) (ASTitem, codeParsingError) {
	// Parse `fn`
	assert(eq(keywords.get().keywordType, Function))
	location := keywords.get().location
	if !keywords.next() {
		return ASTitem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function definition, unexpected end of keywords"),
			location: keywords.get().location,
		}
	}

	// Parse (for example): `b0 returnStutus, b1 = myFunction(b1="test", b2=myVariable)`
	functionHeadAST, err := variableMutationToAST(keywords)
	if err.msg != nil {
		return ASTitem{}, err
	}
	if functionHeadAST.contents[1].itemType != Function {
		return ASTitem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function definition, expected a value of type Function here, got a value of type " + functionHeadAST.contents[1].itemType.String()),
			location: functionHeadAST.contents[1].location,
		}

	}
	err = nextNonEmpty(keywords, "After function head, unexpected end of keywords")
	if err.msg != nil {
		return ASTitem{}, err
	}

	// Parse function body
	functionBodyAST, err := blockToAST(keywords)
	if err.msg != nil {
		return ASTitem{}, err
	}

	// Return
	return ASTitem{
		location: location,
		itemType: Function,
		name:     functionHeadAST.name,
		contents: append(
			functionHeadAST.contents,
			functionBodyAST...,
		),
	}, codeParsingError{}
}

func keywordsToAST(bareKeywordList []keyword) ([]ASTitem, codeParsingError) {
	var ASTitems []ASTitem
	keywords := listIterator[keyword]{
		currentIndex: 0,
		list:         bareKeywordList,
	}
	canHaveImportStatements := true
	for true {
		switch keywords.get().keywordType {
		case Newline, Comment:
		case Import:
			// TODO: Consider syntax choices around import:
			// - Should we force their to only be one import that lists every dependency?
			// - Do we even need imports? We could just automaticaly import things based on the charecters before the period (EG: `std.math.intToString 42`)
			//   - How do we handle overlap, for example if there was a function called std that was defined in a file in the folder?
			if !canHaveImportStatements {
				return []ASTitem{}, codeParsingError{
					msg:      errors.New("All imports must be at the beggining of the file (excluding commas and newlines)."),
					location: keywords.get().location,
				}
			}
			importLocation := keywords.get().location
			if !keywords.next() || keywords.get().keywordType != Name {
				return []ASTitem{}, codeParsingError{
					msg:      errors.New("During the parsing of an import statement, after `import`, unexpected end of file"),
					location: keywords.get().location,
				}
			}
			if keywords.get().keywordType != Name {
				return []ASTitem{}, codeParsingError{
					msg:      errors.New("During the parsing of an import statement, after `import`, expecting keyword of type Name, got a keyword of type " + keywords.get().keywordType.String()),
					location: keywords.get().location,
				}

			}
			add(&ASTitems, ASTitem{
				location: importLocation,
				itemType: Import,
				name:     keywords.get().contents,
			})
		case Function:
			canHaveImportStatements = false
			functionAST, err := functionDefinitionToAST(&keywords)
			if err.msg != nil {
				return []ASTitem{}, err
			}
			add(&ASTitems, functionAST)
		default:
			return []ASTitem{}, codeParsingError{
				msg:      errors.New("Expecting keyword of type `Newline`, `Comment` `Import`, or `Function`. Got a keyword of type `" + keywords.get().keywordType.String() + "`."),
				location: keywords.get().location,
			}
		}
		if !keywords.next() {
			break
		}
	}
	return ASTitems, codeParsingError{}
}
