package main

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// Parser.go
// =========
// Responsible for parsing a list of keywords that make up a common assembly
// file into a list of the `topLevelASTitem` type that is defined in `AST.go`
// while reporting syntax errors in the list of keywords that were not detected
// by `convertFileIntoParsedCode`.

func nextNonEmpty(keywords *listIterator[keyword], errorMsg string) codeParsingError {
	for true {
		if !keywords.next() {
			return codeParsingError{
				msg:          errors.New(errorMsg),
				textLocation: keywords.get().location,
			}
		}
		if keywords.get().keywordType != Newline &&
			keywords.get().keywordType != Comment {
			return codeParsingError{}
		}
	}
	panic("Unreachable")
}

func splitCondition(keywords []keyword, splitType keywordType) ([][]keyword, codeParsingError) {
	lastSplitIndex := -1
	nesting := 0
	returnValue := [][]keyword{}
	for index, item := range keywords {
		switch item.keywordType {
		case IncreaseNesting:
			if item.contents != "(" {
				return [][]keyword{}, codeParsingError{
					msg:          errors.New("In conditions, keyword of type IncreaseNesting must have contents (, got a keyword with contents " + item.contents),
					textLocation: item.location,
				}
			}
			nesting++
		case DecreaseNesting:
			if item.contents != ")" {
				return [][]keyword{}, codeParsingError{
					msg:          errors.New("In conditions, keyword of type DecreaseNesting must have contents ), got a keyword with contents " + item.contents),
					textLocation: item.location,
				}
			}
			nesting--
			if nesting < 0 {
				return [][]keyword{}, codeParsingError{
					msg:          errors.New("Unexpected ), without a ( before it to match"),
					textLocation: item.location,
				}
			}
		case splitType:
			if nesting > 0 {
				continue
			}
			if index == 0 || index == len(keywords)-1 {
				return [][]keyword{}, codeParsingError{
					msg:          errors.New("Cannot have And or Or at the start or end of a condition"),
					textLocation: item.location,
				}
			}
			add(&returnValue, keywords[lastSplitIndex+1:index])
			lastSplitIndex = index
		}
	}
	if nesting > 0 {
		return [][]keyword{}, codeParsingError{
			msg:          errors.New("In condition, there is " + fmt.Sprint(nesting) + " unclosed ("),
			textLocation: keywords[0].location,
		}
	}
	add(&returnValue, keywords[lastSplitIndex+1:])
	return returnValue, codeParsingError{}
}

func parseCondition(keywords []keyword) (condition, codeParsingError) {
	// Handle outside brackets
	if keywords[0].contents == "(" && keywords[len(keywords)-1].contents == ")" {
		return parseCondition(keywords[1 : len(keywords)-1])
	}

	// Handle conditions with `and` in them
	andClauses, err := splitCondition(keywords, And)
	if err.msg != nil {
		return nil, err
	}
	if len(andClauses) > 1 {
		return parseConditionClauses(andClauses, true)
	}

	// Handle conditions with `or` in them
	orClauses, err := splitCondition(keywords, Or)
	if err.msg != nil {
		return nil, err
	}
	if len(orClauses) > 1 {
		return parseConditionClauses(orClauses, false)
	}

	// Handle if there is only 1 keyword
	if len(keywords) == 1 {
		if keywords[0].keywordType != BoolValue {
			return nil, codeParsingError{
				msg:          errors.New("Expect conditions that only have one keyword to be of type BoolValue, got a keyword of type `" + keywords[0].keywordType.String() + "`"),
				textLocation: keywords[0].location,
			}
		}
		assert(or(eq(keywords[0].contents, "false"),
			eq(keywords[0].contents, "true")))
		return booleanValue{
			textLocation: keywords[0].location,
			value:        keywords[0].contents == "true",
		}, codeParsingError{}
	}

	// Handle comparisons without `and` or `or` in them
	return parseComparison(keywords)
}

// Converts a string to a register
func stringToRegister(in string) Register {
	assert(eq(in[0], 'r'))
	register, err := strconv.Atoi(in[1:])
	assert(eq(err, nil))
	return Register(register)
}

// Parses function arguments. This logic is also used to parse the values in return statements.
// After a succesful execution of this function, `keywords.get()` returns the `)` for functions or
// `}` for returns at the end of the arguments.
func parseFunctionArguments(keywords *listIterator[keyword]) ([]registerAndRawValueAndLocation, codeParsingError) {
	if keywords.get().keywordType == DecreaseNesting {
		return []registerAndRawValueAndLocation{}, codeParsingError{}
	}
	functionArguments := []registerAndRawValueAndLocation{}
	for true {
		register := UnkownRegister
		if keywords.get().keywordType == RegisterKeyword {
			// Parse register
			register = stringToRegister(keywords.get().contents)
			err := nextNonEmpty(keywords, "After function argument register, unexpected end of keywords")
			if err.msg != nil {
				return []registerAndRawValueAndLocation{}, err
			}

			// Parse =
			if keywords.get().keywordType != Assignment || keywords.get().contents != "=" {
				return []registerAndRawValueAndLocation{}, codeParsingError{
					textLocation: keywords.get().location,
					msg:          errors.New("Expected keyword of type VariableMutation with contents =, got `" + keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
				}
			}
			err = nextNonEmpty(keywords, "While parsing function arguments, after =, unexpected end of keywords")
			if err.msg != nil {
				return []registerAndRawValueAndLocation{}, err
			}
		}

		// Parse value
		valueAST, err := parseRawValue(keywords)
		if err.msg != nil {
			return []registerAndRawValueAndLocation{}, err
		}
		err = nextNonEmpty(keywords, "After value, unexpected end of keywords")
		if err.msg != nil {
			return []registerAndRawValueAndLocation{}, err
		}

		// Append to arguments
		add(&functionArguments, registerAndRawValueAndLocation{
			textLocation: valueAST.location(),
			register:     register,
			value:        valueAST,
		})

		// Parse ,
		if keywords.get().keywordType != ListSyntax {
			return functionArguments, codeParsingError{}
		}
		err = nextNonEmpty(keywords, "After `,`, unexpected end of keywords")
		if err.msg != nil {
			return nil, err
		}
	}
	panic("Unreachable")
}

// After a succsesful execution of this function, `keywords.get()` should return the keyword at the
// end of the variable value. This keyword should always be of type Name.
func parseVariableValue(keywords *listIterator[keyword]) (variableValue, codeParsingError) {
	out := variableValue{
		textLocation:             keywords.get().location,
		variableIsDropped:        false,
		pointerDereferenceLayers: 0,
	}
	for true {
		switch keywords.get().keywordType {
		case Dereference:
			out.pointerDereferenceLayers++
		case DropVariable:
			if out.variableIsDropped {
				return variableValue{}, codeParsingError{
					msg:          errors.New("This variable is already dropped"),
					textLocation: keywords.get().location,
				}
			} else {
				out.variableIsDropped = true
			}
		case Name:
			out.name = keywords.get().contents
			return out, codeParsingError{}
		default:
			return variableValue{}, codeParsingError{
				msg:          errors.New("During parsing of variable value: Expected a keyword of type Name, DropVariable, or Dereference. Got a keyword of type " + keywords.get().keywordType.String()),
				textLocation: keywords.get().location,
			}
		}
		err := nextNonEmpty(keywords, "After a keyword of type DropVariable or Dereference, unexpected end of keywords")
		if err.msg != nil {
			return variableValue{}, err
		}
	}
	panic("Unreachable")
}

func parseRawValue(keywords *listIterator[keyword]) (rawValue, codeParsingError) {
	switch keywords.get().keywordType {
	case Name, DropVariable, Dereference:
		return parseVariableValue(keywords)
	case PositiveInteger:
		number, err := strconv.ParseUint(keywords.get().contents, 10, 64)
		assert(eq(err, nil))
		return numberValue[uint64]{
			textLocation: keywords.get().location,
			value:        number,
		}, codeParsingError{}
	case NegativeInteger:
		number, err := strconv.ParseInt(keywords.get().contents, 10, 64)
		assert(eq(err, nil))
		return numberValue[int64]{
			textLocation: keywords.get().location,
			value:        number,
		}, codeParsingError{}
	case Decimal:
		number, err := strconv.ParseFloat(keywords.get().contents, 64)
		assert(eq(err, nil))
		return numberValue[float64]{
			textLocation: keywords.get().location,
			value:        number,
		}, codeParsingError{}
	case CharValue:
		assert(eq(keywords.get().contents[0], '\''))
		assert(eq(keywords.get().contents[len(keywords.get().contents)-1], '\''))
		return charecterValue{
			textLocation: keywords.get().location,
			value:        keywords.get().contents[1 : len(keywords.get().contents)-1],
		}, codeParsingError{}
	case StringValue:
		assert(eq(keywords.get().contents[0], '"'))
		assert(eq(keywords.get().contents[len(keywords.get().contents)-1], '"'))
		return stringValue{
			textLocation: keywords.get().location,
			value:        keywords.get().contents[1 : len(keywords.get().contents)-1],
		}, codeParsingError{}
	default:
		return nil, codeParsingError{
			msg:          errors.New("While parsing raw value, unexpected keyword type " + keywords.get().keywordType.String() + " expecting a keyword of type Name, Decimal, NegativeInteger, PositiveInteger, FloatNumber, StringValue, CharValue, Dereference, or DropVariable"),
			textLocation: keywords.get().location,
		}
	}

}

// Parses a chained comparison into an AST node. `keywords` cannot contain a keyword
// where `contents == "and"` or `contents == "or"`, or else this function will
// panic. If `keywords` may contain a keyword where
// `contents == "and"` or `contents == "or"`, then use `conditionToAST`.
func parseComparison(keywordList []keyword) (condition, codeParsingError) {
	assert(greaterThan(len(keywordList), 0))

	var comparisonType byte
	unchainedComparisons := []condition{}
	keywords := listIterator[keyword]{list: keywordList}

	for true {
		comparisonFirstArg, err := parseRawValue(&keywords)
		if err.msg != nil {
			return nil, err
		}
		if !keywords.next() {
			if keywords.currentIndex == 0 {
				return nil, codeParsingError{
					textLocation: keywords.get().location,
					msg:          errors.New("Unexpected end of comparison, expecting either >, >=, <, <=, ==, or !="),
				}
			} else if len(unchainedComparisons) == 1 {
				return unchainedComparisons[0], codeParsingError{}
			} else {
				return boolean{
					isAndInsteadOfOr: true,
					conditions:       unchainedComparisons,
					textLocation:     keywordList[0].location,
				}, codeParsingError{}
			}
		}

		comparisonKeyword := keywords.get()
		if comparisonKeyword.keywordType != ComparisonSyntax {
			return nil, codeParsingError{
				textLocation: comparisonKeyword.location,
				msg:          errors.New("Expecting a keyword of type ComparisonSyntax (==, !=, >, <, >=, <=), got a keyword of type " + comparisonKeyword.keywordType.String() + "."),
			}
		}
		if comparisonType == 0 {
			// If this is the first iteration of the loop, then set comparisonType
			comparisonType = comparisonKeyword.contents[0]
			if comparisonType != '=' && comparisonType != '!' &&
				comparisonType != '<' && comparisonType != '>' {
				panic("Unexpected internal state: `comparisonToAST` got a keyword of type ComparisonSyntax with " +
					"contents `" +
					comparisonKeyword.contents +
					"` expecting the keyword contents to start with either =, !, <, >.")
			}
		} else {
			// If comparisonType is already set to a non-zero value, then this is after the first itereation
			// of the for loop, and we should check that the comparison is valid given comparisonType
			if comparisonType != comparisonKeyword.contents[0] {
				return nil, codeParsingError{
					textLocation: comparisonKeyword.location,
					msg:          errors.New("Expecting comparisons in greatness chain to match"),
				}
			}
			if comparisonType == '!' {
				return nil, codeParsingError{
					textLocation: comparisonKeyword.location,
					msg:          errors.New("You cannot chain comparisons of type !"),
				}
			}
		}

		err = nextNonEmpty(&keywords, "During parsing of comparison, after `"+keywords.get().contents+
			"`, unexpected end of keywords. Expected a value.")
		if err.msg != nil {
			return nil, err
		}

		comparisonSecondArg, err := parseRawValue(&keywords)
		if err.msg != nil {
			return nil, err
		}

		comparisonOperation := UnknownComparisonOperation
		switch comparisonKeyword.contents {
		case ">":
			comparisonOperation = GreaterThan
		case "<":
			comparisonOperation = LessThan
		case ">=":
			comparisonOperation = GreaterThanOrEqual
		case "<=":
			comparisonOperation = LessThanOrEqual
		case "==":
			comparisonOperation = Equal
		case "!=":
			comparisonOperation = NotEqual
		}

		add(&unchainedComparisons, condition(comparison{
			operator:     comparisonOperation,
			textLocation: comparisonKeyword.location,
			leftValue:    comparisonFirstArg,
			rightValue:   comparisonSecondArg,
		}))
	}
	panic("Unreachable")
}

func parseConditionClauses(
	clauses [][]keyword,
	createAndBooleanInsteadOfOr bool,
) (boolean, codeParsingError) {
	conditionClauses := make([]condition, len(clauses))
	for i, clause := range clauses {
		err := codeParsingError{}
		conditionClauses[i], err = parseCondition(clause)
		if err.msg != nil {
			return boolean{}, err
		}
	}
	return boolean{
		conditions:       conditionClauses,
		isAndInsteadOfOr: createAndBooleanInsteadOfOr,
		textLocation:     clauses[0][0].location,
	}, codeParsingError{}
}

// Parses a "conditional block" into an AbstractSyntaxTree node. This consists
// of ignoring the first keyword, then parsing a condition, then parsing a
// block. After a succsesful execution of this function, keywords.get().contents
// should equal to "}"
func parseConditionalBlock(keywords *listIterator[keyword]) (textLocation, condition, []statement, codeParsingError) {
	// Save the location to return later
	location := keywords.get().location

	// Ignore the first keyword
	if !keywords.next() {
		return textLocation{}, nil, nil, codeParsingError{
			msg:          errors.New("During parsing of the conditonal block, unexpected end of keywords slice."),
			textLocation: keywords.get().location,
		}
	}

	// Get the keywords in the condition
	conditionKeywords := []keyword{}
	for keywords.get().contents != "{" {
		add(&conditionKeywords, *keywords.get())
		if !keywords.next() {
			return textLocation{}, nil, nil, codeParsingError{
				msg:          errors.New("Unexpected end of keywords."),
				textLocation: keywords.get().location,
			}
		}
	}

	// Parse the condition into AST
	condition, err := parseCondition(conditionKeywords)
	if err.msg != nil {
		return textLocation{}, nil, nil, err
	}

	// Parse the block into AST
	block, err := parseBlock(keywords)
	if err.msg != nil {
		return textLocation{}, nil, nil, err
	}

	// Return
	return location, condition, block, codeParsingError{}
}

// After a succsesful execution of this function, keywords.get().contents should equal to "}"
func parseIfElseStatement(keywords *listIterator[keyword]) (ifElseStatement, codeParsingError) {
	// Parse if block
	out := ifElseStatement{}
	err := codeParsingError{}
	out.textLocation, out.condition, out.ifBlock, err = parseConditionalBlock(keywords)
	if err.msg != nil {
		return ifElseStatement{}, err
	}

	// Parse else block if there is one
	if keywords.currentIndex+1 < len(keywords.list) {
		if keywords.list[keywords.currentIndex+1].contents == "elif" {
			assert(eq(keywords.next(), true))
			elseBlockStatement, err := parseIfElseStatement(keywords)
			if err.msg != nil {
				return ifElseStatement{}, err
			}
			out.elseBlock = []statement{elseBlockStatement}
		} else if keywords.list[keywords.currentIndex+1].contents == "else" {
			assert(eq(keywords.next(), true))
			if !keywords.next() {
				return ifElseStatement{}, codeParsingError{
					msg:          errors.New("Unexpected end of keywords. Either remove the else, or add a block after the else."),
					textLocation: keywords.get().location,
				}
			}
			err := codeParsingError{}
			out.elseBlock, err = parseBlock(keywords)
			if err.msg != nil {
				return ifElseStatement{}, err
			}
		}
	}

	// Return
	return out, codeParsingError{}
}

// After a succsesful execution of this function, keywords.get().contents should equal to "}"
func parseBlock(keywords *listIterator[keyword]) ([]statement, codeParsingError) {
	// Parse {
	if keywords.get().contents != "{" {
		return nil, codeParsingError{
			msg:          errors.New("Expecting { to start a new block."),
			textLocation: keywords.get().location,
		}
	}

	ASTitems := []statement{}

	// Parse each statement inside the block
	for true {
		err := nextNonEmpty(keywords, "During the parsing of a block, unexpected end of the keywords slice")
		if err.msg != nil {
			return nil, err
		}
		switch keywords.get().keywordType {
		case FunctionReturn:
			// Save the location of the return
			location := keywords.get().location

			// Move past the return keyword
			err := nextNonEmpty(keywords, "Unexpected end of keywords")
			if err.msg != nil {
				return nil, err
			}

			// Parse the return values
			returnValues, err := parseFunctionArguments(keywords)
			if err.msg != nil {
				return nil, err
			}

			// Parse }
			if keywords.get().keywordType != DecreaseNesting || keywords.get().contents != "}" {
				return nil, codeParsingError{
					textLocation: keywords.get().location,
					msg:          errors.New("Expected keyword of type DecreaseNesting with contents `)`, got `" + keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
				}
			}

			// Return the function
			return append(ASTitems, returnStatement{
				textLocation:   location,
				returnedValues: returnValues,
			}), codeParsingError{}

		case RegisterKeyword, Dereference, Name:
			variableMutationAST, err := parseMutationStatement(keywords)
			if err.msg != nil {
				return nil, err
			}
			add(&ASTitems, statement(variableMutationAST))

		case DropVariable:
			location := keywords.get().location
			err := nextNonEmpty(keywords, "Unexpected end of keywords in drop statement, expected a variable name.")
			if err.msg != nil {
				return nil, err
			}
			if keywords.get().keywordType != Name {
				return nil, codeParsingError{
					msg: errors.New("Got a keyword of type " + keywords.get().keywordType.String() + " in a drop statement. Expected a variable name."),
				}
			}
			add(&ASTitems, statement(dropVariableStatement{
				textLocation: location,
				variable:     keywords.get().contents,
			}))

		// Statements that start with control flow syntax can either be a while loop
		// or an `if`, `elif`, `else` statement.
		case IfStatement:
			conditionalBlock, err := parseIfElseStatement(keywords)
			if err.msg != nil {
				return nil, err
			}
			add(&ASTitems, statement(conditionalBlock))
		case WhileLoop:
			loop := whileLoop{}
			err := codeParsingError{}
			loop.textLocation, loop.condition, loop.loopBody, err = parseConditionalBlock(keywords)
			if err.msg != nil {
				return nil, err
			}
			add(&ASTitems, statement(loop))
		case BreakStatement:
			add(&ASTitems, statement(breakStatement(keywords.get().location)))
		case ContinueStatement:
			add(&ASTitems, statement(continueStatement(keywords.get().location)))

		// The only valid statement that starts with decrease nesting is }, which exits the block scope
		case DecreaseNesting:
			switch keywords.get().contents {
			case "}":
				return ASTitems, codeParsingError{}
			default:
				return nil, codeParsingError{
					msg:          errors.New("Expecting a keyword of type `DecreaseNesting` within a block to have contents `}` got `" + keywords.get().contents + "`."),
					textLocation: keywords.get().location,
				}
			}
		default:
			return nil, codeParsingError{
				msg:          errors.New("Expecting a keyword of type Newline, Comment, Name, ControlFlowSyntax, Register, or DecreaseNesting, got a keyword of type " + keywords.get().keywordType.String()),
				textLocation: keywords.get().location,
			}
		}
	}
	panic("Unreachable")
}

// After a succsesful execution of this function, `keywords.get()` should return
// the keyword after the end of the mutation destination.
func parseVariableMutationDestination(keywords *listIterator[keyword]) ([]variableMutationDestination, codeParsingError) {
	out := []variableMutationDestination{}
	for true {
		current := variableMutationDestination{register: UnkownRegister, textLocation: keywords.get().location}

		if keywords.get().keywordType == RegisterKeyword {
			current.register = stringToRegister(keywords.get().contents)
			err := nextNonEmpty(keywords, "While parsing the destination for a variable mutation, after register, unexpected end of keywords")
			if err.msg != nil {
				return nil, err
			}
		}

		if keywords.get().keywordType == Dereference || keywords.get().keywordType == Name {
			variable, err := parseVariableValue(keywords)
			if err.msg != nil {
				return nil, err
			}
			if variable.variableIsDropped {
				return nil, codeParsingError{
					msg:          errors.New("While parsing the destination for a variable mutation, the variable in the destination is dropped. This means that mutating it would be useless."),
					textLocation: variable.location(),
				}
			}
			current.name = variable.name
			current.pointerDereferenceLayers = variable.pointerDereferenceLayers
			err = nextNonEmpty(keywords, "While parsing the destination for a variable mutation, after name, unexpected end of keywords")
			if err.msg != nil {
				return nil, err
			}
		}

		if current.register == -1 && current.name == "" {
			return nil, codeParsingError{
				msg:          errors.New("While parsing the destination for a variable mutation, expected a keyword of type RegisterKeyword, Name, or Dereference. Got a keyword of type " + keywords.get().keywordType.String()),
				textLocation: keywords.get().location,
			}
		}

		add(&out, current)

		if keywords.get().keywordType != ListSyntax {
			return out, codeParsingError{}
		}
		err := nextNonEmpty(keywords, "Unexpected end of keywords after ,")
		if err.msg != nil {
			return nil, err
		}
	}
	panic("Unreachable")
}

// Parses a statement starting with a keyword of type Name, Register, DropVariable, or Dereference
// into an AST item. Examples of this type of statement include:
// - `b0 returnStatus, b1 = myFunction(b1="test", b2=myVariable)`
// - `b0 result, b1 = power(b1=base, b2=power)`
// - `^pointsToACharecter = 'a'`
// After a succsesful execution of this function, keywords.get() should return
// the keyword at end of the statement.
func parseMutationStatement(keywords *listIterator[keyword]) (mutationStatement, codeParsingError) {
	// Parse the destination (the things that are being mutated)
	out := mutationStatement{textLocation: keywords.get().location}
	err := codeParsingError{}
	out.destination, err = parseVariableMutationDestination(keywords)
	if err.msg != nil {
		return mutationStatement{}, err
	}

	// Parse =, ++, --, +=, -=, *=, /=, and (if needed) `valueBeingAssignedToVariable`
	mutationOperation := keywords.get().keywordType
	switch mutationOperation {

	default:
		return mutationStatement{}, codeParsingError{
			textLocation: keywords.get().location,
			msg: errors.New("After a variable/register that is being mutated, expected" +
				" a keyword of type Assignment, Increment, Decrement, PlusEquals, " +
				"MinusEquals, MultiplyEquals, or DivideEquals, got `" +
				keywords.get().contents + "` of type " +
				keywords.get().keywordType.String()),
		}

	case Increment:
		out.operation = incrementBy1{keywords.get().location}

	case Decrement:
		out.operation = decrementBy1{keywords.get().location}

	case Assignment, PlusEquals, MinusEquals, MultiplyEquals, DivideEquals:
		// Next keyword
		err = nextNonEmpty(keywords, "After `"+keywords.get().contents+
			"` (variable mutation operator), unexpected end of keywords")
		if err.msg != nil {
			return mutationStatement{}, err
		}

		// Custom parsing of assignment where first keyword of value is of type Name, since that could be a function call
		if mutationOperation == Assignment && keywords.get().keywordType == Name {
			// Parse name
			name := keywords.get()
			oldKeywordsIndex := keywords.currentIndex

			// Early return if this is a variable reference, and not a function call
			if !keywords.next() || keywords.get().keywordType != IncreaseNesting || keywords.get().contents != "(" {
				keywords.currentIndex = oldKeywordsIndex
				out.operation = setToRawValue{val: variableValue{
					name:                     name.contents,
					textLocation:             name.location,
					variableIsDropped:        false,
					pointerDereferenceLayers: 0,
				}}
				return out, codeParsingError{}
			}

			// Parse (
			err := nextNonEmpty(keywords, "After Name and then (, unexpected end of keywords")
			if err.msg != nil {
				return mutationStatement{}, err
			}

			// Parse arguments
			functionArguments, err := parseFunctionArguments(keywords)

			// Parse )
			if keywords.get().keywordType != DecreaseNesting || keywords.get().contents != ")" {
				return mutationStatement{}, codeParsingError{
					textLocation: keywords.get().location,
					msg:          errors.New("Expected keyword of type DecreaseNesting with contents `)`, got `" + keywords.get().contents + "` of type " + keywords.get().keywordType.String()),
				}
			}

			// Set the mutation operation
			out.operation = setToFunctionCallValue{
				textLocation: name.location,
				functionName: name.contents,
				functionArgs: functionArguments,
			}
		} else {
			rawValue, err := parseRawValue(keywords)
			if err.msg != nil {
				return mutationStatement{}, err
			}
			switch mutationOperation {
			case Assignment:
				out.operation = setToRawValue{val: rawValue}
			case PlusEquals:
				out.operation = incrementByRawValue{val: rawValue}
			case MinusEquals:
				out.operation = decrementByRawValue{val: rawValue}
			case MultiplyEquals:
				out.operation = multiplyByRawValue{val: rawValue}
			case DivideEquals:
				out.operation = divideByRawValue{val: rawValue}
			}
		}

	}
	return out, codeParsingError{}
}

// After a succsesful execution of this function, keywords.get().contents should equal to "}"
func parseFunctionDefinition(keywords *listIterator[keyword]) (functionDefinition, codeParsingError) {
	out := functionDefinition{}

	// Parse `fn`
	assert(eq(keywords.get().keywordType, Function))
	out.textLocation = keywords.get().location
	err := nextNonEmpty(keywords, "During the parsing of a function definition, unexpected end of keywords")
	if err.msg != nil {
		return functionDefinition{}, err
	}

	// TODO: HACK: We currently just parse a mutation statement and than convert
	// it into the functions args and mutated registers in order to parse a
	// function head.
	// Parse (for example): `b0 returnStutus, b1 = myFunction(b1="test", b2=myVariable)`
	mutationStatement, err := parseMutationStatement(keywords)
	if err.msg != nil {
		return functionDefinition{}, err
	}
	functionCall, isSetToFunctionCallValue := mutationStatement.operation.(setToFunctionCallValue)
	if !isSetToFunctionCallValue {
		return functionDefinition{}, codeParsingError{
			textLocation: mutationStatement.textLocation,
			msg: errors.New("Expected the mutation operation of a function head to be " +
				"setToFunctionCallValue, but got " +
				fmt.Sprint(reflect.TypeOf(mutationStatement.operation))),
		}
	}
	out.mutatedRegisters = make([]registerAndNameAndLocation, len(mutationStatement.destination))
	for i, mutatedItem := range mutationStatement.destination {
		if mutatedItem.pointerDereferenceLayers > 0 {
			return functionDefinition{}, codeParsingError{
				msg:          errors.New("Expected the pointer dereference layers of a mutated register to be 0, got " + fmt.Sprint(mutatedItem.pointerDereferenceLayers)),
				textLocation: mutatedItem.textLocation,
			}
		}
		if mutatedItem.register == UnkownRegister {
			return functionDefinition{}, codeParsingError{
				msg:          errors.New("Expected the register to be specified"),
				textLocation: mutatedItem.textLocation,
			}
		}
		out.mutatedRegisters[i] = registerAndNameAndLocation{
			register:     mutatedItem.register,
			name:         mutatedItem.name,
			textLocation: mutatedItem.textLocation,
		}
	}
	out.name = functionCall.functionName
	out.arguments = make([]registerAndNameAndLocation, len(functionCall.functionArgs))
	for i, argument := range functionCall.functionArgs {
		if argument.register == UnkownRegister {
			return functionDefinition{}, codeParsingError{
				msg:          errors.New("Expected the register to be specified"),
				textLocation: argument.textLocation,
			}
		}
		variableValue, ok := argument.value.(variableValue)
		if !ok {
			return functionDefinition{}, codeParsingError{
				msg:          errors.New("Expected a variable value"),
				textLocation: argument.location(),
			}
		}
		if variableValue.pointerDereferenceLayers > 0 {
			return functionDefinition{}, codeParsingError{
				msg:          errors.New("Expected the pointer dereference layers of an argument to be 0, got " + fmt.Sprint(variableValue.pointerDereferenceLayers)),
				textLocation: variableValue.textLocation,
			}
		}
		if variableValue.variableIsDropped == true {
			return functionDefinition{}, codeParsingError{
				msg:          errors.New("Expected the function arg to not be dropped"),
				textLocation: variableValue.textLocation,
			}
		}
		out.arguments[i].register = argument.register
		out.arguments[i].name = variableValue.name
		out.arguments[i].textLocation = argument.textLocation
	}
	err = nextNonEmpty(keywords, "After function head, unexpected end of keywords")
	if err.msg != nil {
		return functionDefinition{}, err
	}

	// Parse function body
	out.body, err = parseBlock(keywords)
	if err.msg != nil {
		return functionDefinition{}, err
	}

	// Return
	return out, codeParsingError{}
}

func parseTopLevelASTitems(bareKeywordList []keyword) ([]topLevelASTitem, codeParsingError) {
	var ASTitems []topLevelASTitem
	keywords := listIterator[keyword]{
		currentIndex: 0,
		list:         bareKeywordList,
	}
	for true {
		switch keywords.get().keywordType {
		case Newline, Comment:
		case Import:
			// TODO: Design and implement the ability to import other common assembly files:
			// - Should we force their to only be one import per file that lists every dependency?
			// - Do we even need imports? We could just automaticaly import things based on the charecters before the period (EG: `std.math.intToString 42`)
			//   - How do we handle overlap, for example if there was a function called std that was defined in a file in the folder?
			return nil, codeParsingError{
				msg:          errors.New("Import statements are not supported yet"),
				textLocation: keywords.get().location,
			}
		case Function:
			functionAST, err := parseFunctionDefinition(&keywords)
			if err.msg != nil {
				return nil, err
			}
			add(&ASTitems, topLevelASTitem(functionAST))
		default:
			return nil, codeParsingError{
				msg:          errors.New("Expecting keyword of type `Newline`, `Comment` `Import`, or `Function`. Got a keyword of type `" + keywords.get().keywordType.String() + "`."),
				textLocation: keywords.get().location,
			}
		}
		if !keywords.next() {
			break
		}
	}
	return ASTitems, codeParsingError{}
}
