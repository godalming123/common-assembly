package main

import (
	"errors"
	"fmt"
	"strings"
)

// Parser.go
// =========
// Responsible for parsing a list of keywords that make up a common assembly
// file into a list of the `AbstractSyntaxTreeItem` type below while reporting
// any syntax errors in the list of keywords that were not detected by
// `convertFileIntoParsedCode`.

////////////////////////////////////
// ABSTRACT SYNTAX TREE ITEM TYPE //
////////////////////////////////////

//go:generate stringer -type=possibleAbstractSyntaxTreeItem
type possibleAbstractSyntaxTreeItem uint8

const (
	Undefined possibleAbstractSyntaxTreeItem = iota
	ImportStatement
	WhileLoop
	BreakStatement
	ContinueStatement
	IfStatement
	ElseStatement
	FunctionDefintion
	FunctionArgs
	FunctionArg
	Return
	ArgMutatability
	ArgRegister
	FunctionCall
	AssemblySyscall

	ValueComparison  // >, <, >=, <=, ==, !=
	BooleanLogic     // and, or
	VariableMutation // mov, add, sub, mul, div, mod

	PointerOfVariable
	String
	Char
	Int
	Float
	VarName
	Bool
)

type AbstractSyntaxTreeItem struct {
	itemType possibleAbstractSyntaxTreeItem
	name     string
	contents []AbstractSyntaxTreeItem
	location textLocation
}

func (AST AbstractSyntaxTreeItem) print(indentation int) {
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

//////////////////////////////////////////////////////////////////////////
// FUNCTIONS TO CONVERT KEYWORDS INTO AN ABSTRACT SYNTAX TREE ITEM LIST //
//////////////////////////////////////////////////////////////////////////

func conditionToAST(keywords []keyword) (AbstractSyntaxTreeItem, codeParsingError) {
	// TODO: Support () in comparisons to specify order of operations
	// Handle conditions with `and` in them
	andClauses := splitSlice(keywords, func(item keyword) bool {
		return item.contents == "and"
	})
	if len(andClauses) > 1 {
		return conditionClausesToAST(andClauses, "and")
	}

	// Handle conditions with `or` in them
	orClauses := splitSlice(keywords, func(item keyword) bool {
		return item.contents == "or"
	})
	if len(orClauses) > 1 {
		return conditionClausesToAST(orClauses, "or")
	}

	// Handle if there is only 1 keyword
	if len(keywords) == 1 {
		if keywords[0].keywordType != BoolValue {
			return AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("Expect conditions that only have one keyword to be of type BoolValue, got a keyword of type `" + keywords[0].keywordType.String() + "`"),
				location: keywords[0].location,
			}
		}
		assert(keywords[0].contents == "true" || keywords[0].contents == "false")
		return AbstractSyntaxTreeItem{
			name:     keywords[0].contents,
			itemType: Bool,
			location: keywords[0].location,
		}, codeParsingError{}
	}

	// Handle comparisons without `and` or `or` in them
	return comparisonToAST(keywords)
}

func parseValue(keyword keyword) (AbstractSyntaxTreeItem, codeParsingError) {
	var ASTitemType possibleAbstractSyntaxTreeItem
	switch keyword.keywordType {
	case Name:
		ASTitemType = VarName
	case IntNumber:
		ASTitemType = Int
	case FloatNumber:
		ASTitemType = Float
	case StringValue:
		ASTitemType = String
		assert(keyword.contents[0] == '"' && keyword.contents[len(keyword.contents)-1] == '"')
		keyword.contents = keyword.contents[1 : len(keyword.contents)-1]
	case CharValue:
		ASTitemType = Char
		assert(keyword.contents[0] == '\'' && keyword.contents[len(keyword.contents)-1] == '\'')
		keyword.contents = keyword.contents[1 : len(keyword.contents)-1]
	case PointerOf:
		ASTitemType = PointerOfVariable
		assert(keyword.contents[0] == '^')
		keyword.contents = keyword.contents[1:]
	default:
		return AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("While parsing value, unexpected keyword type " + keyword.keywordType.String() + " expecting a keyword of type Name, IntNumber, FloatNumber, StringValue, CharValue, or PointerOf"),
			location: keyword.location,
		}
	}
	return AbstractSyntaxTreeItem{
		name:     keyword.contents,
		itemType: ASTitemType,
		location: keyword.location,
	}, codeParsingError{}
}

func nextNonEmpty(keywords *listIterator[keyword]) bool {
	for true {
		if !keywords.next() {
			return false
		}
		if keywords.get().keywordType != Newline &&
			keywords.get().keywordType != Comment {
			return true
		}
	}
	panic("Unreachable")
}

// Parses a comparison into an AST node. `keywords` cannot contain a keyword
// where `contents == "and" || contents == "or"`, or else this function will
// panic. If `keywords` may contain a keyword where
// `contents == "and" || contents == "or"`, then use `conditionToAST`.
func comparisonToAST(keywordList []keyword) (AbstractSyntaxTreeItem, codeParsingError) {
	assert(len(keywordList) > 0)

	var comparisonType byte
	comparisonClauses := []AbstractSyntaxTreeItem{}
	keywords := listIterator[keyword]{list: keywordList}

	for true {
		comparisonFirstArg, err := parseValue(*keywords.get())
		if err.msg != nil {
			return AbstractSyntaxTreeItem{}, err
		}

		if !nextNonEmpty(&keywords) {
			if keywords.currentIndex == 0 {
				return AbstractSyntaxTreeItem{}, codeParsingError{
					location: keywords.get().location,
					msg:      errors.New("Unexpected end of comparison, expecting either >, >=, <, <=, ==, or !="),
				}
			} else if len(comparisonClauses) == 1 {
				return comparisonClauses[0], codeParsingError{}
			} else {
				return AbstractSyntaxTreeItem{
					itemType: BooleanLogic,
					name:     "and",
					contents: comparisonClauses,
					location: keywordList[0].location,
				}, codeParsingError{}
			}
		}

		if comparisonType == 0 {
			comparisonType = keywords.get().contents[0]
			if comparisonType != '=' && comparisonType != '!' &&
				comparisonType != '<' && comparisonType != '>' {
				panic("Unexpected internal state: `comparisonToAST` got a keyword with " +
					"contents `" +
					keywords.get().contents +
					"` expecting the keyword contents to start with either =, !, <, >.")
			}
		} else {
			if comparisonType != keywords.get().contents[0] {
				return AbstractSyntaxTreeItem{}, codeParsingError{
					location: keywords.get().location,
					msg:      errors.New("Expecting comparisons in greatness chain to match"),
				}
			}
			if comparisonType == '!' {
				return AbstractSyntaxTreeItem{}, codeParsingError{
					location: keywords.get().location,
					msg:      errors.New("You cannot chain comparisons of type !"),
				}
			}
		}

		if !nextNonEmpty(&keywords) {
			return AbstractSyntaxTreeItem{}, codeParsingError{
				location: keywords.get().location,
				msg:      errors.New("Unexpected end of comparison, expecting value"),
			}
		}

		comparisonSecondArg, err := parseValue(*keywords.get())
		if err.msg != nil {
			return AbstractSyntaxTreeItem{}, err
		}

		add(&comparisonClauses, AbstractSyntaxTreeItem{
			itemType: ValueComparison,
			name:     keywordList[keywords.currentIndex-1].contents,
			location: keywordList[keywords.currentIndex-1].location,
			contents: []AbstractSyntaxTreeItem{
				comparisonFirstArg,
				comparisonSecondArg,
			},
		})
	}
	panic("Unreachable")
}

func conditionClausesToAST(clauses [][]keyword, conditionFunctionName string) (AbstractSyntaxTreeItem, codeParsingError) {
	conditionClausesAST := []AbstractSyntaxTreeItem{}
	for _, clause := range clauses {
		clauseASTitem, err := conditionToAST(clause)
		if err.msg != nil {
			return AbstractSyntaxTreeItem{}, err
		}
		add(&conditionClausesAST, clauseASTitem)
	}
	return AbstractSyntaxTreeItem{
		location: clauses[0][0].location,
		itemType: BooleanLogic,
		name:     conditionFunctionName,
		contents: conditionClausesAST,
	}, codeParsingError{}
}

// Parses a "conditional block" into an AbstractSyntaxTree node. This consists
// of ignoring the first keyword, then parsing a condition, then parsing a
// block, and then concatonating the result into one AST item of type
// `ASTitemType`.
func conditionalBlockToAST(keywords *listIterator[keyword], ASTitemType possibleAbstractSyntaxTreeItem) (AbstractSyntaxTreeItem, codeParsingError) {
	// Save the location
	location := keywords.get().location

	// Ignore the first keyword
	if !keywords.next() {
		return AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During parsing of the conditonal block, unexpected end of keywords slice."),
			location: keywords.get().location,
		}
	}

	// Get the keywords in the condition
	conditionKeywords := []keyword{}
	for keywords.get().contents != "{" {
		add(&conditionKeywords, *keywords.get())
		if !keywords.next() {
			return AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("Unexpected end of keywords."),
				location: keywords.get().location,
			}
		}
	}

	// Parse the condition into AST
	conditionAST, err := conditionToAST(conditionKeywords)
	if err.msg != nil {
		return AbstractSyntaxTreeItem{}, err
	}

	// Parse the block into AST
	blockAST, err := blockToAST(keywords)
	if err.msg != nil {
		return AbstractSyntaxTreeItem{}, err
	}

	// Return
	return AbstractSyntaxTreeItem{
		itemType: ASTitemType,
		contents: append([]AbstractSyntaxTreeItem{conditionAST}, blockAST...),
		location: location,
	}, codeParsingError{}
}

func ifStatementToAST(keywords *listIterator[keyword]) (AbstractSyntaxTreeItem, codeParsingError) {
	// Parse if block
	out, err := conditionalBlockToAST(keywords, IfStatement)
	if err.msg != nil {
		return AbstractSyntaxTreeItem{}, err
	}

	// Parse else block if there is one
	if keywords.currentIndex+1 < len(keywords.list) {
		if keywords.list[keywords.currentIndex+1].contents == "elif" {
			assert(keywords.next())
			elifBlock, err := ifStatementToAST(keywords)
			if err.msg != nil {
				return AbstractSyntaxTreeItem{}, err
			}
			add(&out.contents, AbstractSyntaxTreeItem{
				location: elifBlock.location,
				itemType: ElseStatement,
				contents: []AbstractSyntaxTreeItem{elifBlock},
			})
		} else if keywords.list[keywords.currentIndex+1].contents == "else" {
			assert(keywords.next())
			if !keywords.next() {
				return AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Unexpected end of block. Either remove the else, or add a block after the else."),
					location: keywords.get().location,
				}
			}
			elseBlockContents, err := blockToAST(keywords)
			if err.msg != nil {
				return AbstractSyntaxTreeItem{}, err
			}
			add(&out.contents, AbstractSyntaxTreeItem{
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
func blockToAST(keywords *listIterator[keyword]) ([]AbstractSyntaxTreeItem, codeParsingError) {
	// Parse {
	if keywords.get().contents != "{" {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("Expecting { to start a new block."),
			location: keywords.get().location,
		}
	}

	ASTitems := []AbstractSyntaxTreeItem{}

	// Parse each statement inside the block
	for nextNonEmpty(keywords) {
		switch keywords.get().keywordType {
		case FunctionReturn:
			if nextNonEmpty(keywords) && keywords.get().contents == "}" {
				return append(ASTitems, AbstractSyntaxTreeItem{
					location: keywords.list[keywords.currentIndex-1].location,
					itemType: Return,
				}), codeParsingError{}
			}
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				location: keywords.get().location,
				msg:      errors.New("After return, next keyword must be } to close the scope"),
			}

		case Syscall:
			add(&ASTitems, AbstractSyntaxTreeItem{
				location: keywords.get().location,
				itemType: AssemblySyscall,
			})

		// Statements that start with a name can either be a function call, a function definition, or a variable mutation
		case Name:
			location := keywords.get().location
			name := keywords.get()
			if !nextNonEmpty(keywords) {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of statement that starts with a name, unexpected end of keywords slice."),
					location: keywords.get().location,
				}
			}
			switch keywords.get().keywordType {
			case VariableModificationSyntax:
				// Parse function call
				functionCall := ""
				switch keywords.get().contents {
				case "+=", "++":
					functionCall = "add"
				case "-=", "--":
					functionCall = "sub"
				case "*=":
					// TODO: Optimize: Use `add name, name` instead of `mul name, 2`
					functionCall = "mul"
				case "/=":
					functionCall = "div"
				case "%=":
					functionCall = "mod"
				case "=":
					// TODO: Optimize: Use xor instead of move if the value is being set to 0
					// becuase it is quicker
					functionCall = "mov"
				default:
					panic("Unexpected internal state: `stringToKeywords.go` should not produce a keyword of type VariableModifucationSyntax that has contents other then `+=`, `++`, `-=`, `--`, `*=`, `/=`, `%=`, or `=`.")
				}
				// Parse function first value
				functionValue1 := AbstractSyntaxTreeItem{
					location: keywords.get().location,
				}
				switch keywords.get().contents {
				case "++", "--":
					functionValue1.itemType = Int
					functionValue1.name = "1"
				case "=", "+=", "-=", "*=", "/=", "%=":
					if !nextNonEmpty(keywords) {
						return []AbstractSyntaxTreeItem{}, codeParsingError{
							msg:      errors.New("After " + keywords.get().contents + ", unexpected end of keywords."),
							location: keywords.get().location,
						}
					}
					// TODO: Maybe we should support defining a variable as the result of a comparison
					if keywords.get().keywordType == PointerOf {
						if functionCall != "mov" {
							return []AbstractSyntaxTreeItem{}, codeParsingError{
								location: keywords.get().location,
								msg:      errors.New("During parsing of variable mutation, ^ can only be used with ="),
							}
						}
						assert(keywords.get().contents[0] == '^')
						functionCall = "movq"
						functionValue1.itemType = VarName
						functionValue1.name = keywords.get().contents[1:]
					} else {
						var err codeParsingError
						functionValue1, err = parseValue(*keywords.get())
						if err.msg != nil {
							return []AbstractSyntaxTreeItem{}, err
						}
					}
				}
				// Parse function second value
				functionValue2, err := parseValue(*name)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				// Append to ASTitems
				add(&ASTitems, AbstractSyntaxTreeItem{
					location: location,
					itemType: VariableMutation,
					name:     functionCall,
					contents: []AbstractSyntaxTreeItem{
						functionValue1,
						functionValue2,
					},
				})
			case VariableCreationSyntax:
				if !keywords.next() {
					return []AbstractSyntaxTreeItem{}, codeParsingError{
						msg:      errors.New("During the parsing of statement that starts with a name, and then has a VariableCreationSyntax, unexpected end of keywords slice."),
						location: keywords.get().location,
					}
				}
				function, err := functionValueToAST(keywords)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, AbstractSyntaxTreeItem{
					location: location,
					itemType: FunctionDefintion,
					name:     name.contents,
					contents: function,
				})
			case Mutatability:
				// `keywords.get()` is the first keyword in the arguments of a function call
				functionArgsASTitems, err := functionArgumentsToAST(keywords, false)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, AbstractSyntaxTreeItem{
					location: location,
					itemType: FunctionCall,
					name:     name.contents,
					contents: functionArgsASTitems,
				})
			default:
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg: errors.New(
						"During the parsing of a statement that starts with a name, expecting " +
							"either VariableModifuctaionSyntax, VariableCreationSyntax, or " +
							"Mutatability as the keyword type that comes after the name. Got `" +
							keywords.get().keywordType.String() +
							"`.",
					),
					location: keywords.get().location,
				}
			}

		// Statements that start with control flow syntax can either be a while loop
		// or an `if`, `elif`, `else` statement.
		case ControlFlowSyntax:
			switch keywords.get().contents {
			case "if":
				conditionalBlock, err := ifStatementToAST(keywords)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, conditionalBlock)

			case "while":
				conditionalBlock, err := conditionalBlockToAST(keywords, WhileLoop)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, conditionalBlock)

			case "break":
				add(&ASTitems, AbstractSyntaxTreeItem{location: keywords.get().location, itemType: BreakStatement})

			case "continue":
				add(&ASTitems, AbstractSyntaxTreeItem{location: keywords.get().location, itemType: ContinueStatement})

			case "else", "elif":
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Unexpected " + keywords.get().contents + " statement. Else/elif statements must go directly after the end of an if/elif block."),
					location: keywords.get().location,
				}

			default:
				panic("Unexpected internal state: Got a keyword of type ControlFlowSyntax with contents `" + keywords.get().contents + "` expected a keyword of this type to have contents `if`, `elif`, `else`, `while`, `break`, or `continue`.")
			}

		// The only valid statement that starts with decrease nesting is }, which exits the block scope
		case DecreaseNesting:
			switch keywords.get().contents {
			case "}":
				return ASTitems, codeParsingError{}
			default:
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Expecting a keyword of type `DecreaseNesting` within a block to have contents `}` got `" + keywords.get().contents + "`."),
					location: keywords.get().location,
				}
			}
		default:
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("Expecting a keyword of type Newline, Comment, Name, ControlFlowSyntax, or DecreaseNesting, got a keyword of type " + keywords.get().keywordType.String()),
				location: keywords.get().location,
			}
		}
	}

	return []AbstractSyntaxTreeItem{}, codeParsingError{
		msg:      errors.New("During the parsing of a block, unexpected end of the keywords slice."),
		location: keywords.get().location,
	}
}

// Parses function args in a function call or definition. Expects
// `keywords.get()` to be just after the `(` of a function definition.
// `keywords.get()` is set to the `)` of a function definition.
func functionArgumentsToAST(keywords *listIterator[keyword], argumentsHaveRegisters bool) ([]AbstractSyntaxTreeItem, codeParsingError) {
	functionArgs := []AbstractSyntaxTreeItem{}
	for true {
		// Save the location of the function arg
		argLocation := keywords.get().location

		// Parse argument mutatability
		if keywords.get().keywordType != Mutatability {
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("During the parsing of function args, expect a mutatability (arg/mut/mutArg) first. Note: All functions must have at least one argument in order for them to actually be able to do anything."),
				location: keywords.get().location,
			}
		}
		argMutatability := keywords.get().contents
		if !keywords.next() {
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("During the parsing of function args, unexpected end of the keywords slice."),
				location: keywords.get().location,
			}
		}

		// Parse argument variable name
		// TODO: Common assembly code would be much more readable and modifiabile if inline values could be passed to function calls instead of having to pass names
		if keywords.get().keywordType != Name {
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg: errors.New(
					"During the parsing of function args, after the mutatability of a " +
						"function arg, expecting a keyword of type name, got a keyword of type " +
						keywords.get().keywordType.String() + fmt.Sprint(keywords.get().location),
				),
				location: keywords.get().location,
			}
		}
		argVariableName := keywords.get().contents
		if !keywords.next() {
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("During the parsing of function args, unexpected end of the keywords slice."),
				location: keywords.get().location,
			}
		}

		// Add the function arg to the list of functionArgs
		add(&functionArgs, AbstractSyntaxTreeItem{
			itemType: FunctionArg,
			name:     argVariableName,
			location: argLocation,
			contents: []AbstractSyntaxTreeItem{
				{
					itemType: ArgMutatability,
					location: argLocation,
					name:     argMutatability,
				},
			},
		})

		// Parse argument register if `argumentsHaveRegisters` is true
		if argumentsHaveRegisters {
			if keywords.get().keywordType != Register {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of function args for function definition, after the name of a function arg, expect a register."),
					location: keywords.get().location,
				}
			}
			add(&functionArgs[len(functionArgs)-1].contents, AbstractSyntaxTreeItem{
				itemType: ArgRegister,
				name:     keywords.get().contents,
				location: keywords.get().location,
			})
			if !keywords.next() {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of function args, unexpected end of the keywords slice."),
					location: keywords.get().location,
				}
			}
		}

		// Parse the `,`
		if keywords.get().keywordType != ListSyntax {
			break
		}
		if !keywords.next() {
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("During the parsing of function args, unexpected end of the keywords slice."),
				location: keywords.get().location,
			}
		}
	}
	return functionArgs, codeParsingError{}
}

func functionValueToAST(keywords *listIterator[keyword]) ([]AbstractSyntaxTreeItem, codeParsingError) {
	// Parse function arguments
	if keywords.get().contents != "(" {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function value, got " + keywords.get().contents + " expecting ( as first keyword."),
			location: keywords.get().location,
		}
	}
	if !keywords.next() {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function value, unexpected end of the keywords slice."),
			location: keywords.get().location,
		}
	}
	functionArguments, err := functionArgumentsToAST(keywords, true)
	if err.msg != nil {
		return []AbstractSyntaxTreeItem{}, err
	}
	if keywords.get().contents != ")" {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function value, after function arguments expecting )."),
			location: keywords.get().location,
		}
	}
	if !keywords.next() {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function value, unexpected end of the keywords slice."),
			location: keywords.get().location,
		}
	}

	// Parse function body
	functionBody, err := blockToAST(keywords)
	if err.msg != nil {
		return []AbstractSyntaxTreeItem{}, err
	}

	// Return
	return append(
			[]AbstractSyntaxTreeItem{
				{
					location: functionArguments[0].location,
					itemType: FunctionArgs,
					contents: functionArguments,
				},
			},
			functionBody...,
		),
		codeParsingError{}
}

func keywordsToAST(bareKeywordList []keyword) ([]AbstractSyntaxTreeItem, codeParsingError) {
	var ASTitems []AbstractSyntaxTreeItem
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
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("All imports must be at the beggining of the file (excluding commas and newlines)."),
					location: keywords.get().location,
				}
			}
			importLocation := keywords.get().location
			if !keywords.next() || keywords.get().keywordType != Name {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of an import statement, unexpected end of file after `import`. Expecting keyword of type Name."),
					location: keywords.get().location,
				}
			}
			add(&ASTitems, AbstractSyntaxTreeItem{
				location: importLocation,
				itemType: ImportStatement,
				name:     keywords.get().contents,
			})
		case Name:
			canHaveImportStatements = false

			// Save location
			functionLocation := keywords.get().location

			// Parse function name
			functionName := keywords.get().contents
			if !nextNonEmpty(&keywords) {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of a function, unexpected end of keywords slice."),
					location: keywords.get().location,
				}
			}

			// Parse ::
			if keywords.get().contents != "::" {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Unexpected `" + keywords.get().contents + "`, during the parsing of a function, after function name expecting `::`"),
					location: keywords.get().location,
				}
			}
			if !nextNonEmpty(&keywords) {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of a function, unexpected end of the keywords slice."),
					location: keywords.get().location,
				}
			}

			// Parse function value
			function, err := functionValueToAST(&keywords)
			if err.msg != nil {
				return []AbstractSyntaxTreeItem{}, err
			}
			add(&ASTitems, AbstractSyntaxTreeItem{
				location: functionLocation,
				itemType: FunctionDefintion,
				name:     functionName,
				contents: function,
			})
		default:
			return []AbstractSyntaxTreeItem{}, codeParsingError{
				msg:      errors.New("Expecting keyword of type `Newline`, `Comment`, `BuiltInFunction`, or `Name`. Got a keyword of type `" + keywords.get().keywordType.String() + "`."),
				location: keywords.get().location,
			}
		}
		if !keywords.next() {
			break
		}
	}
	return ASTitems, codeParsingError{}
}
