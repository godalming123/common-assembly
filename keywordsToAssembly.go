package main

import (
	"errors"
	"fmt"
	"strings"
)

// ConvertKeywordsToAssembly.go
// ============================
// Responsible for parsing a list of keywords that make up a common assembly
// file into a string of assembly instructions for a specefic architecture,
// while reporting any syntax errors in the list of keywords that were not
// detected by `convertFileIntoParsedCode`.

type functionArg struct {
	codeName     string
	assemblyName string
}

type functionProperties struct {
	// The name of the function, set to `anonymous` if it is an anonymous function
	name string

	// The number of function args
	argsNum int

	// Values that are parsed to the function that are not changed when the
	// function is ran, but do affect the result of the function
	constantFunctionArgs []functionArg

	// Values that are parsed to the function that are changed when the function
	// is ran, and do affect the result of the function
	mutateableFunctionArgs []functionArg

	// Values that are parsed to the function that are changed when the function
	// is ran, but do not affect the result of the function
	mutateableFunctionMemory []functionArg
}

type Architecture uint8

const (
	wasm Architecture = iota
	amd64
	arm64
)

type OS uint8

const (
	linux OS = iota
	windows
	web
)

type keywordParsingState struct {
	currentKeyword               int
	numberOfControlFlowJumpNames int
	targetArchitecture           Architecture
	targetOS                     OS
	keywords                     []keyword
	inlineValues                 []string
	dataSectionText              string
	functionStack                []functionProperties
}

func (parsingState *keywordParsingState) getCurrentFunctionProperties() *functionProperties {
	return &parsingState.functionStack[len(parsingState.functionStack)-1]
}

func (parsingState *keywordParsingState) getCurrentKeyword() *keyword {
	return &parsingState.keywords[parsingState.currentKeyword]
}

func findAssemblyNameFromCodeName(codeName string, functionArgListList [][]functionArg) string {
	for _, functionArgList := range functionArgListList {
		for _, functionArg := range functionArgList {
			if functionArg.codeName == codeName {
				return functionArg.assemblyName
			}
		}
	}
	return ""
}

// Parses a value into valid assembly, sets parsingState.currentKeyword to the keyword at the end of the value
func (parsingState *keywordParsingState) parseValueIntoAssembly() (string, codeParsingError) {
	switch parsingState.getCurrentKeyword().keywordType {
	case Name:
		return findAssemblyNameFromCodeName(
			parsingState.keywords[parsingState.currentKeyword].contents,
			[][]functionArg{
				parsingState.getCurrentFunctionProperties().mutateableFunctionArgs,
				parsingState.getCurrentFunctionProperties().mutateableFunctionMemory,
			},
		), codeParsingError{}
	case BoolValue:
		return parsingState.getCurrentKeyword().contents, codeParsingError{}
	case StringValue:
		add(&parsingState.inlineValues, "\""+parsingState.getCurrentKeyword().contents+"\"")
		return "INLINEVAR" + fmt.Sprint(len(parsingState.inlineValues)), codeParsingError{}
	case Number:
		// TODO: Handle float numbers, negetives, numbers that aren't base 10, and numbers with _ in them
		return parsingState.getCurrentKeyword().contents, codeParsingError{}
	}
	return "", codeParsingError{
		msg: errors.New(
			"During parsing of value, got unexpected keyword type " +
				convertKeywordTypeToString(parsingState.getCurrentKeyword().keywordType) +
				". The keyword contents is " +
				parsingState.getCurrentKeyword().contents,
		),
		line:   parsingState.getCurrentKeyword().line,
		column: parsingState.getCurrentKeyword().column,
	}
}

// Parses a statement starting with a functioncall into the IR.
func (parsingState *keywordParsingState) parseStatementStartingWithFunctionCallIntoIr() codeParsingError {
	// TODO: Add support for different architectures
	// TODO: Optimize: If the function is only called once, then the pointer to the current instruction does not need to be pushed to the stack
	parsingState.dataSectionText += "call "
	parsingState.dataSectionText += parsingState.getCurrentKeyword().contents
	for parsingState.currentKeyword < len(parsingState.keywords) {
		parsingState.currentKeyword++
		switch parsingState.getCurrentKeyword().keywordType {
		case Name, BoolValue, StringValue, Number:
			// TODO:
			// If there is a function definition like so:
			// function :: (a mutArg rsi) {...}
			// And it is called like so:
			// function 69
			// Then calling it would IMPLICITLY modify rsi, and so that needs to be disallowed
			value, err := parsingState.parseValueIntoAssembly()
			if err.msg != nil {
				return err
			}
			parsingState.dataSectionText += value
		case ListSyntax:
			parsingState.dataSectionText += "__"
		case Newline:
			// TODO: Add the assembly code after the `call` function which is used to reset the stack
			parsingState.dataSectionText += "\n"
			return codeParsingError{}
		}
	}
	return codeParsingError{
		msg:    errors.New("During parsing function call, unexpected end of file"),
		line:   parsingState.getCurrentKeyword().line,
		column: parsingState.getCurrentKeyword().column,
	}
}

func (parsingState *keywordParsingState) parseStatementStartingWithControlFlowSyntaxIntoIr() codeParsingError {
	parsingState.numberOfControlFlowJumpNames++
	switch parsingState.getCurrentKeyword().contents {
	case "while":
		// TODO: Add support for break
		parsingState.currentKeyword++
		jumpName := "CONTROL_FLOW_JUMP" + fmt.Sprint(parsingState.numberOfControlFlowJumpNames)
		parsingState.dataSectionText += jumpName
		parsingState.dataSectionText += ":"
		if parsingState.getCurrentKeyword().contents == "true" {
			parsingState.currentKeyword++
			err := parsingState.parseBlockIntoIr()
			if err.msg != nil {
				return err
			}
			parsingState.dataSectionText += "goto " + jumpName
		} else if parsingState.getCurrentKeyword().contents == "false" {
			// TODO: Warn when there is skipped code
		}
	case "if":
	case "else":
	}
	return codeParsingError{}
}

func (parsingState *keywordParsingState) parseStatementStartingWithNameIntoIr() codeParsingError {
	assemblyName := findAssemblyNameFromCodeName(
		parsingState.getCurrentKeyword().contents,
		[][]functionArg{
			parsingState.getCurrentFunctionProperties().mutateableFunctionArgs,
			parsingState.getCurrentFunctionProperties().mutateableFunctionMemory,
		},
	)
	println(assemblyName)
	println(parsingState.getCurrentKeyword().line)
	println(parsingState.getCurrentKeyword().column)
	if assemblyName == "" {
		return parsingState.parseStatementStartingWithFunctionCallIntoIr()
	}
	parsingState.currentKeyword++
	switch parsingState.getCurrentKeyword().contents {
	case "+=", "++":
		parsingState.dataSectionText += "add " // TODO: Change the assembly added for different architecures
	case "-=", "--":
		parsingState.dataSectionText += "sub "
	case "*=":
		parsingState.dataSectionText += "mul " // TODO: Use `add name, name` instaed of `mul name, 2`
	case "/=":
		parsingState.dataSectionText += "div "
	case "=":
		parsingState.dataSectionText += "mov " // TODO: Use xor instead of move if the value is being set to 0 becuase it is quicker
	default:
		return codeParsingError{
			msg:    errors.New("After mutateable name, expecting =, +=, -=, *=, /=, ++, --"),
			line:   parsingState.getCurrentKeyword().line,
			column: parsingState.getCurrentKeyword().column,
		}
	}
	parsingState.dataSectionText += assemblyName
	parsingState.dataSectionText += ", "
	switch parsingState.getCurrentKeyword().contents {
	case "++", "--":
		parsingState.dataSectionText += "1\n"
	case "=", "+=", "-=", "*=", "/=":
		parsingState.currentKeyword++
		value, err := parsingState.parseValueIntoAssembly()
		if err.msg != nil {
			return err
		}
		parsingState.dataSectionText += value
		parsingState.dataSectionText += "\n"
	}
	parsingState.currentKeyword++
	return codeParsingError{}
}

func (parsingState *keywordParsingState) parseBlockIntoIr() codeParsingError {
	// Parse {
	if parsingState.getCurrentKeyword().contents != "{" {
		return codeParsingError{
			msg:    errors.New("Expecting {"),
			line:   parsingState.getCurrentKeyword().line,
			column: parsingState.getCurrentKeyword().column,
		}
	}

	// Parse each statement inside the block
	parsingState.currentKeyword++
	for parsingState.currentKeyword < len(parsingState.keywords) {
		err := codeParsingError{}
		switch parsingState.getCurrentKeyword().keywordType {
		case Name:
			err = parsingState.parseStatementStartingWithNameIntoIr()
		case ControlFlowSyntax:
			err = parsingState.parseStatementStartingWithControlFlowSyntaxIntoIr()
		case DecreaseNesting:
			switch parsingState.getCurrentKeyword().contents {
			case "}":
				parsingState.currentKeyword++
				return codeParsingError{}
			case "]", ")":
				return codeParsingError{
					msg: errors.New(
						"Unexpected nesting decrease without a nesting increase " +
							parsingState.getCurrentKeyword().contents,
					),
					line:   parsingState.getCurrentKeyword().line,
					column: parsingState.getCurrentKeyword().column,
				}
			}
		case BuiltInFunction:
			switch parsingState.getCurrentKeyword().contents {
			case "import":
				return codeParsingError{
					msg:    errors.New("`import` keyword cannot be used inside {}"),
					line:   parsingState.getCurrentKeyword().line,
					column: parsingState.getCurrentKeyword().column,
				}
			case "syscall":
				parsingState.currentKeyword++
				parsingState.dataSectionText += "syscall\n"
			}
		case Comment:
		case Newline:
		default:
			return codeParsingError{
				msg: errors.New(
					"During parsing of code block, got unexpected keyword type " +
						convertKeywordTypeToString(parsingState.getCurrentKeyword().keywordType) +
						". The keyword contents is " +
						parsingState.getCurrentKeyword().contents,
				),
				line:   parsingState.getCurrentKeyword().line,
				column: parsingState.getCurrentKeyword().column,
			}
		}
		if err.msg != nil {
			return err
		}
		parsingState.currentKeyword++
	}

	return codeParsingError{
		msg:    errors.New("Unexpected end of scope, try adding }"),
		line:   parsingState.getCurrentKeyword().line,
		column: parsingState.getCurrentKeyword().column,
	}
}

func (parsingState *keywordParsingState) parseFunctionDefinitionIntoIr() codeParsingError {
	// Parse function name
	if parsingState.getCurrentKeyword().keywordType != Name {
		return codeParsingError{
			msg:    errors.New("Expecting the start of a function definition. The first keyword of a function defintion should be the name of the function."),
			line:   parsingState.getCurrentKeyword().line,
			column: parsingState.getCurrentKeyword().column,
		}
	}
	parsingState.dataSectionText += parsingState.getCurrentKeyword().contents // TODO: Change this code for different architecutures
	add(&parsingState.functionStack, functionProperties{
		name: parsingState.getCurrentKeyword().contents,
	})
	parsingState.currentKeyword++
	if parsingState.getCurrentKeyword().contents != "::" {
		return codeParsingError{
			msg:    errors.New("Second keyword of function definition should always be ::"),
			line:   parsingState.getCurrentKeyword().line,
			column: parsingState.getCurrentKeyword().column,
		}
	}
	parsingState.currentKeyword++
	if parsingState.getCurrentKeyword().contents != "(" {
		return codeParsingError{
			msg:    errors.New("Third keyword of function definition should always be ("),
			line:   parsingState.getCurrentKeyword().line,
			column: parsingState.getCurrentKeyword().column,
		}
	}
	parsingState.currentKeyword++

	// Parse function args
	for true {
		if parsingState.getCurrentKeyword().keywordType != Name {
			break
		}
		argumentProps := functionArg{
			codeName: parsingState.getCurrentKeyword().contents,
		}
		parsingState.currentKeyword++
		argumentType := parsingState.getCurrentKeyword().contents
		parsingState.currentKeyword++
		if parsingState.getCurrentKeyword().keywordType == Name {
			argumentProps.assemblyName = parsingState.getCurrentKeyword().contents
			parsingState.currentKeyword++
		} else {
			argumentProps.assemblyName = "ARG" + fmt.Sprint(parsingState.getCurrentFunctionProperties().argsNum) + "_TYPE_" + strings.ToUpper(argumentType)
			parsingState.dataSectionText += "_"
			parsingState.dataSectionText += argumentProps.assemblyName
		}
		switch argumentType {
		case "mut":
			add(&parsingState.getCurrentFunctionProperties().mutateableFunctionMemory, argumentProps)
		case "arg":
			add(&parsingState.getCurrentFunctionProperties().mutateableFunctionMemory, argumentProps)
		case "mutArg":
			add(&parsingState.getCurrentFunctionProperties().mutateableFunctionMemory, argumentProps)
		default:
			return codeParsingError{
				msg:    errors.New("After function argument name, expecting `mut`, `arg`, or `mutArg`."),
				line:   parsingState.getCurrentKeyword().line,
				column: parsingState.getCurrentKeyword().column,
			}
		}
		parsingState.getCurrentFunctionProperties().argsNum++
		if parsingState.getCurrentKeyword().contents != "," {
			break
		}
		parsingState.currentKeyword++
	}
	if parsingState.getCurrentKeyword().contents != ")" {
		return codeParsingError{
			msg:    errors.New("Unexpected keyword. Expecting , or ) after function arg."),
			line:   parsingState.getCurrentKeyword().line,
			column: parsingState.getCurrentKeyword().column,
		}
	}
	parsingState.dataSectionText += ":\n"
	parsingState.currentKeyword++

	// Parse function body
	return parsingState.parseBlockIntoIr()
}

func (parsingState *keywordParsingState) parseFromBeginneng() codeParsingError {
	for parsingState.currentKeyword < len(parsingState.keywords) {
		currentKeyword := parsingState.getCurrentKeyword()
		switch currentKeyword.keywordType {
		case Name:
			err := parsingState.parseFunctionDefinitionIntoIr()
			if err.msg != nil {
				return err
			}
		case BuiltInFunction:
			// TODO: support import statements, or error for syscalls
		case Newline, Comment:
			parsingState.currentKeyword++
		default:
			return codeParsingError{
				msg: errors.New(
					"During parsing of main file, got unexpected keyword type " +
						convertKeywordTypeToString(parsingState.getCurrentKeyword().keywordType) +
						". The keyword contents is " +
						parsingState.getCurrentKeyword().contents,
				),
				line:   parsingState.getCurrentKeyword().line,
				column: parsingState.getCurrentKeyword().column,
			}
		}
	}
	return codeParsingError{}
}
