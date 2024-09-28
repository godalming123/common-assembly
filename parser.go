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
	IfStatement
	ElifStatement
	ElseStatement
	FunctionDefintion
	FunctionArgs
	FunctionArg
	ArgMutatability
	ArgRegister
	FunctionCall
	AssemblyFunctionCall
	ComparisonValue
	GreatnessChainItemsCanBeEqual    // <=, >=
	GreatnessChainItemsCannotBeEqual // <, >
)

type AbstractSyntaxTreeItem struct {
	itemType possibleAbstractSyntaxTreeItem
	name     string
	contents []AbstractSyntaxTreeItem
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

	// Handle comparisons without `and` or `or` in them
	return comparisonToAST(keywords)
}

// Parses a comparison into an AST node. `keywords` cannot contain a keyword
// where `contents == "and" || contents == "or"`, or else this function will
// panic. If `keywords` may contain a keyword where
// `contents == "and" || contents == "or"`, then use `conditionToAST`.
func comparisonToAST(keywords []keyword) (AbstractSyntaxTreeItem, codeParsingError) {
	const equalCmp = '='
	const notEqualCmp = '!'
	const ascendingCmp = '<'
	const descendingCmp = '>'
	var comparisonType byte

	comparisonContentsAST := []AbstractSyntaxTreeItem{}
	comparisonIsValidKeyword := false
	for _, keyword := range keywords {
		if keyword.keywordType == ComparisonSyntax {
			if !comparisonIsValidKeyword {
				return AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Unexpected comparison, comparisons can only go between values that are being compared."),
					location: keyword.location,
				}
			}

			if comparisonType == 0 {
				// If `comparisonType` is 0, then we set `comparisonType` to something other
				// then 0.
				comparisonType = keyword.contents[0]
				if comparisonType != equalCmp &&
					comparisonType != notEqualCmp &&
					comparisonType != ascendingCmp &&
					comparisonType != descendingCmp {
					panic("Unexpected internal state: `comparisonToAST` expects all keywords of type `ComparisonSyntax` to start with either `=`, `!`, `<`, `>`, since they cannot be `and`/`or`.")
				}
			} else if comparisonType == notEqualCmp {
				// Because of the above comment, if `comparisonType` is not 0, then this is
				// after the first `keyword` of type `ComparisonSyntax` in `keywords`,
				// therefore this is a chain. Not equal comparisons cannot be chained.
				return AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Not equal comparisons cannot be chained."),
					location: keyword.location,
				}
			}

			switch comparisonType {
			case ascendingCmp, descendingCmp:
				switch keyword.contents {
				case "<=", ">=":
					add(&comparisonContentsAST, AbstractSyntaxTreeItem{itemType: GreatnessChainItemsCanBeEqual})
				case "<", ">":
					add(&comparisonContentsAST, AbstractSyntaxTreeItem{itemType: GreatnessChainItemsCannotBeEqual})
				default:
					return AbstractSyntaxTreeItem{}, codeParsingError{
						msg:      errors.New("While parsing comparison of type greatness chain, expecting either <=, >=, <, > as comparison keywords. Got: `" + keyword.contents + "`."),
						location: keyword.location,
					}
				}
			case equalCmp:
				if keyword.contents != "==" {
					return AbstractSyntaxTreeItem{}, codeParsingError{
						msg:      errors.New("Every comparison in equal chain should be `==`"),
						location: keyword.location,
					}
				}
			default:
				panic("Unexpected internal state")
			}
			comparisonIsValidKeyword = false
		} else if keyword.keywordType != Newline && keyword.keywordType != Comment {
			// TODO: Do more parsing of the values between the comparisons
			add(&comparisonContentsAST, AbstractSyntaxTreeItem{
				itemType: ComparisonValue,
				name:     keyword.contents,
			})
			comparisonIsValidKeyword = true
		}
	}

	comparisonFunction := string(comparisonType)
	if comparisonFunction == "=" || comparisonFunction == "!" {
		comparisonFunction += "="
	}

	return AbstractSyntaxTreeItem{
		itemType: AssemblyFunctionCall,
		name:     comparisonFunction,
		contents: comparisonContentsAST,
	}, codeParsingError{}
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
		itemType: AssemblyFunctionCall,
		name:     conditionFunctionName,
		contents: conditionClausesAST,
	}, codeParsingError{}
}

// Parses a "conditional block" into an AbstractSyntaxTree node. This consists
// of ignoring the first keyword, then parsing a condition, then parsing a
// block, and then concatonating the result into one AST item of type
// `ASTitemType`.
func conditionalBlockToAST(keywords *listIterator[keyword], ASTitemType possibleAbstractSyntaxTreeItem) (AbstractSyntaxTreeItem, codeParsingError) {
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
	}, codeParsingError{}
}

func blockToAST(keywords *listIterator[keyword]) ([]AbstractSyntaxTreeItem, codeParsingError) {
	// Parse {
	if keywords.get().contents != "{" {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("Expecting { to start a new block."),
			location: keywords.get().location,
		}
	}

	ASTitems := []AbstractSyntaxTreeItem{}

	// Parse each statment inside the block
	for keywords.next() {
		switch keywords.get().keywordType {

		// Do nothing for newlines, and comments
		case Newline, Comment:

		case Syscall:
			add(&ASTitems, AbstractSyntaxTreeItem{
				itemType: AssemblyFunctionCall,
				name:     "syscall",
			})

		// Statements that start with a name can either be a function call, a function definition, or a variable mutation
		case Name:
			name := keywords.get().contents
			if !keywords.next() {
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
					functionCall = "mul" // TODO: Optimize: Use `add name, name` instaed of `mul name, 2`
				case "/=":
					functionCall = "div"
				case "%=":
					functionCall = "mod"
				case "=":
					functionCall = "mov" // TODO: Optimize: Use xor instead of move if the value is being set to 0 becuase it is quicker
				default:
					panic("Unexpected internal state: `stringToKeywords.go` should not produce a keyword of type VariableModifucationSyntax that has contents other then `+=`, `++`, `-=`, `--`, `*=`, `/=`, `%=`, or `=`.")
				}
				// Parse function value
				functionValue := ""
				switch keywords.get().contents {
				case "++", "--":
					functionValue = "1"
				case "=":
					if !keywords.next() {
						return []AbstractSyntaxTreeItem{}, codeParsingError{
							msg:      errors.New("After =, unexpected end of keywords."),
							location: keywords.get().location,
						}
					}
					switch keywords.get().keywordType {
					// TODO: Maybe we should support defining a variable as the result of a comparison
					case StringValue, BoolValue, IntNumber, FloatNumber:
						functionValue = keywords.get().contents
					default:
						return []AbstractSyntaxTreeItem{}, codeParsingError{
							msg:      errors.New("After =, next keyword must be of type StringValue, BoolValue, IntNumber or FloatNumber. Got a keyword of type `" + keywords.get().keywordType.String() + "`."),
							location: keywords.get().location,
						}
					}
				case "+=", "-=", "*=", "/=", "%=":
					if !keywords.next() {
						return []AbstractSyntaxTreeItem{}, codeParsingError{
							msg:      errors.New("After +=, -=, *=, /=, or %=, unexpected end of keywords."),
							location: keywords.get().location,
						}
					}
					switch keywords.get().keywordType {
					// TODO: Maybe we should support defining a variable as the result of a comparison
					case IntNumber, FloatNumber, Name:
						functionValue = keywords.get().contents
					default:
						return []AbstractSyntaxTreeItem{}, codeParsingError{
							msg:      errors.New("After +=, -=, *=, /=, or %=, next keyword must be of type IntNumber, FloatNumber, or Name. Got a keyword of type `" + keywords.get().keywordType.String() + "`."),
							location: keywords.get().location,
						}
					}
				}
				// Append to ASTitems
				add(&ASTitems, AbstractSyntaxTreeItem{
					itemType: AssemblyFunctionCall,
					name:     functionCall,
					contents: []AbstractSyntaxTreeItem{
						{
							itemType: FunctionArg,
							name:     name,
						},
						{
							itemType: FunctionArg,
							name:     functionValue,
						},
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
					itemType: FunctionDefintion,
					name:     name,
					contents: function,
				})
			case Mutatability:
				// `keywords.get()` is the first keyword in the arguments of a function call
				functionArgsASTitems, err := functionArgumentsToAST(keywords, false)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, AbstractSyntaxTreeItem{
					itemType: FunctionCall,
					name:     name,
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
				conditionalBlock, err := conditionalBlockToAST(keywords, IfStatement)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, conditionalBlock)
				if !keywords.next() {
					return []AbstractSyntaxTreeItem{}, codeParsingError{
						msg:      errors.New("Unexpected end of block. Try adding '}'"),
						location: keywords.get().location,
					}
				}
				for keywords.get().contents == "elif" {
					conditionalBlock, err := conditionalBlockToAST(keywords, ElifStatement)
					if err.msg != nil {
						return []AbstractSyntaxTreeItem{}, err
					}
					add(&ASTitems, conditionalBlock)
				}
				if keywords.get().contents == "else" {
					if !keywords.next() {
						return []AbstractSyntaxTreeItem{}, codeParsingError{
							msg:      errors.New("Unexpected end of block. Either remove the else, or add a block after the else."),
							location: keywords.get().location,
						}
					}
					blockAST, err := blockToAST(keywords)
					if err.msg != nil {
						return []AbstractSyntaxTreeItem{}, err
					}
					add(&ASTitems, AbstractSyntaxTreeItem{
						itemType: ElseStatement,
						contents: blockAST,
					})
				}

			case "while":
				conditionalBlock, err := conditionalBlockToAST(keywords, WhileLoop)
				if err.msg != nil {
					return []AbstractSyntaxTreeItem{}, err
				}
				add(&ASTitems, conditionalBlock)

			case "else":
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("Unexpected `else` statement. Else statements must go directly after the end of a if/elif block."),
					location: keywords.get().location,
				}

			default:
				panic("Unexpected internal state: `stringToKeywords.go` should not produce a keyword of type ControlFlowSyntax that has contents othen then if, else, or while.")
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

		// Parse argument register if `argumentsHaveRegisters` is true
		var argRegister string
		if argumentsHaveRegisters {
			if keywords.get().keywordType != Register {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of function args for function definition, after the name of a function arg, expect a register."),
					location: keywords.get().location,
				}
			}
			argRegister = keywords.get().contents
			if !keywords.next() {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of function args, unexpected end of the keywords slice."),
					location: keywords.get().location,
				}
			}
		}

		// Add the function arg to the list of functionArgs
		add(&functionArgs, AbstractSyntaxTreeItem{
			itemType: FunctionArg,
			name:     argVariableName,
			contents: []AbstractSyntaxTreeItem{
				{
					itemType: ArgMutatability,
					name:     argMutatability,
				},
				{
					itemType: ArgRegister,
					name:     argRegister,
				},
			},
		})

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
			msg:      errors.New("During the parsing of a function, after function name expecting ( as first keyword."),
			location: keywords.get().location,
		}
	}
	if !keywords.next() {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function, unexpected end of the keywords slice."),
			location: keywords.get().location,
		}
	}
	functionArguments, err := functionArgumentsToAST(keywords, true)
	if err.msg != nil {
		return []AbstractSyntaxTreeItem{}, err
	}
	if keywords.get().contents != ")" {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function, after function arguments expecting )."),
			location: keywords.get().location,
		}
	}
	if !keywords.next() {
		return []AbstractSyntaxTreeItem{}, codeParsingError{
			msg:      errors.New("During the parsing of a function, unexpected end of the keywords slice."),
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
			if !keywords.next() || keywords.get().keywordType != Name {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of an import statement, unexpected end of file after `import`. Expecting keyword of type Name."),
					location: keywords.get().location,
				}
			}
			add(&ASTitems, AbstractSyntaxTreeItem{
				itemType: ImportStatement,
				name:     keywords.get().contents,
			})
		case Name:
			canHaveImportStatements = false

			// Parse function name
			functionName := keywords.get().contents
			if !keywords.next() {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of a function, unexpected end of keywords slice."),
					location: keywords.get().location,
				}
			}

			// Parse ::
			if keywords.get().contents != "::" {
				return []AbstractSyntaxTreeItem{}, codeParsingError{
					msg:      errors.New("During the parsing of a function, after function name expecting ::."),
					location: keywords.get().location,
				}
			}
			if !keywords.next() {
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
