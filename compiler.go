package main

import (
	"errors"
	"fmt"
	"strings"
)

// Compiler.go
// ===========
// Responsible for compiling an abstract syntax tree into assembly

type compiledFunction struct {
	references uint
	jumpLabel  string
	// If jumpLabel == "", then this will have `\` to return from this function,
	// and maybe `/FUNCTION_NAME/` to call other functions. This code will still
	// need to be compiled to assembly.
	assembly string
}

type compilerState struct {
	numberOfJumps              uint
	numberOfItemsInDataSection uint
	dataSection                string
	compiledFunctions          map[string]compiledFunction
}

func (jumpState *compilerState) createNewJumpLabel() string {
	jumpState.numberOfJumps++
	return "jumpLabel" + fmt.Sprint(jumpState.numberOfJumps)
}

func (jumpState *compilerState) createNewDataSectionLabel() string {
	jumpState.numberOfItemsInDataSection++
	return "dataSectionLabel" + fmt.Sprint(jumpState.numberOfItemsInDataSection)
}

//go:generate stringer -type=mutatability
type mutatability uint8

const (
	Arg mutatability = iota
	MutArg
	Mut
)

type functionArg struct {
	codeName     string
	mutatability mutatability
	// TODO: Rethink the register names so that they are not specefic to x86,
	// and are easier to understand for people that come from higher level
	// languages.
	register string
}

// Stores the assembly code to be inserted when a control flow keyword is used.
// If the assembly code is a blank string, then that control flow cannot be used
// in the current scope.
type assemblyForControlFlowKeywords struct {
	continueAssembly string
	breakAssembly    string
}

func (state *compilerState) compileBlockToAssembly(
	block []AbstractSyntaxTreeItem,
	parentFunctionArgs []functionArg,
	siblingFunctions map[string][]AbstractSyntaxTreeItem,
	controlFlowKeywordsAssembly assemblyForControlFlowKeywords,
) (string, codeParsingError) {
	assembly := ""
	for index, statement := range block {
		switch statement.itemType {

		case Return:
			assert(index == len(block)-1)
			assembly += "\n\\"
			return assembly, codeParsingError{}

		case FunctionCall:
			functionCallAssembly, err := state.compileFunctionCall(statement, parentFunctionArgs, siblingFunctions)
			if err.msg != nil {
				return "", err
			}
			assembly += functionCallAssembly

		case AssemblySyscall:
			assembly += "\nsyscall"

		case VariableMutation:
			assert(len(statement.contents) == 2)
			assembly += "\n" + statement.name + " "
			firstArg, err := state.convertValueToAssembly(parentFunctionArgs, statement.contents[0])
			if err.msg != nil {
				return "", err
			}
			secondArg, err := state.convertValueToAssembly(parentFunctionArgs, statement.contents[1])
			if err.msg != nil {
				return "", err
			}
			assert(secondArg[0] < '0' || secondArg[1] > '9') // The first argument must be a variable
			assembly += firstArg + ", " + secondArg

		case WhileLoop:
			// Save jump labels
			loopBodyJumpLabel := state.createNewJumpLabel()
			loopConditionJumpLabel := state.createNewJumpLabel()
			loopEndJumpLabel := state.createNewJumpLabel()

			// Add loop head
			assembly += "\njmp " + loopConditionJumpLabel

			// Add loop body
			assembly += "\n" + loopBodyJumpLabel + ":"
			loopBodyAssembly, err := state.compileBlockToAssembly(statement.contents[1:], parentFunctionArgs, siblingFunctions, assemblyForControlFlowKeywords{
				breakAssembly:    "\njmp" + loopEndJumpLabel,
				continueAssembly: "\njmp" + loopConditionJumpLabel,
			})
			if err.msg != nil {
				return "", err
			}
			assembly += loopBodyAssembly

			// Add loop condition
			assembly += "\n" + loopConditionJumpLabel + ":"
			conditionAssembly, err := state.conditionToAssembly(parentFunctionArgs, statement.contents[0], loopBodyJumpLabel, "")
			if err.msg != nil {
				return "", err
			}
			assembly += conditionAssembly

			// Add loop end
			assembly += "\n" + loopEndJumpLabel + ":"

		case IfStatement:
			elseBlockJumpLabel := state.createNewJumpLabel()
			ifCheck, err := state.conditionToAssembly(parentFunctionArgs, statement.contents[0], "", elseBlockJumpLabel)
			if err.msg != nil {
				return "", err
			}
			assembly += ifCheck
			if statement.contents[len(statement.contents)-1].itemType == ElseStatement {
				ifBody, err := state.compileBlockToAssembly(statement.contents[1:len(statement.contents)-1], parentFunctionArgs, siblingFunctions, controlFlowKeywordsAssembly)
				if err.msg != nil {
					return "", err
				}
				assembly += ifBody
				endJumpLabel := state.createNewJumpLabel()
				assembly += "\njmp " + endJumpLabel
				elseBody, err := state.compileBlockToAssembly(statement.contents[len(statement.contents)-1].contents, parentFunctionArgs, siblingFunctions, controlFlowKeywordsAssembly)
				if err.msg != nil {
					return "", err
				}
				assembly += "\n" + elseBlockJumpLabel + ":"
				assembly += elseBody
				assembly += "\n" + endJumpLabel + ":"
			} else {
				ifBody, err := state.compileBlockToAssembly(statement.contents[1:], parentFunctionArgs, siblingFunctions, controlFlowKeywordsAssembly)
				if err.msg != nil {
					return "", err
				}
				assembly += ifBody
				assembly += "\n" + elseBlockJumpLabel + ":"
			}

		default:
			return "", codeParsingError{
				msg: errors.New("Expecting AST nodes in the body of a function to either " +
					"be of type AssemblySyscall, MathFunctionCall, WhileLoop, IfStatement, " +
					"got an AST node of type `" +
					statement.itemType.String() + "`."),
			}
		}
	}
	return assembly, codeParsingError{}
}

func parseArgMutatability(argMutatability AbstractSyntaxTreeItem) (mutatability, codeParsingError) {
	assert(argMutatability.itemType == ArgMutatability)
	switch argMutatability.name {
	case "mut":
		return Mut, codeParsingError{}
	case "arg":
		return Arg, codeParsingError{}
	case "mutArg":
		return MutArg, codeParsingError{}
	default:
		return 0, codeParsingError{
			location: argMutatability.location,
			msg:      errors.New("Function mutateability must be either `mut`, `mutArg`, or `arg`, got " + argMutatability.name),
		}
	}
}

func (state *compilerState) compileFunctionDefinition(
	functionDefinition []AbstractSyntaxTreeItem,
	functionName string,
	siblingFunctions map[string][]AbstractSyntaxTreeItem,
) codeParsingError {
	// Add the `functionName` key to the compiledFunctions hashmap so that when
	// `compileBlockToAssembly()` calls `compileFunctionCall`, that does not call
	// this function if the function being called is the current function being
	// compiled to stop an infinite loop.
	state.compiledFunctions[functionName] = compiledFunction{}

	// Parse the function arguments
	assert(functionDefinition[0].itemType == FunctionArgs)
	functionArgs := []functionArg{}
	for _, arg := range functionDefinition[0].contents {
		assert(arg.itemType == FunctionArg)
		argMutatability, err := parseArgMutatability(arg.contents[0])
		if err.msg != nil {
			return err
		}
		for _, parsedArg := range functionArgs {
			if parsedArg.codeName == arg.name {
				return codeParsingError{
					location: arg.location,
					msg:      errors.New("Cannot have more then one argument with the same name (" + arg.name + ")"),
				}
			}
			if parsedArg.register == arg.contents[1].name {
				return codeParsingError{
					location: arg.location,
					msg:      errors.New("Cannot have more then one argument with the same register (" + arg.contents[1].name + ")"),
				}
			}
		}
		assert(arg.contents[1].itemType == ArgRegister)
		add(&functionArgs, functionArg{
			codeName:     arg.name,
			mutatability: argMutatability,
			register:     arg.contents[1].name,
		})
	}

	// Compile the function
	assembly, err := state.compileBlockToAssembly(functionDefinition[1:], functionArgs, siblingFunctions, assemblyForControlFlowKeywords{})
	if err.msg != nil {
		return err
	}
	assert(assembly != "")
	if assembly[len(assembly)-1] != '\\' {
		// If the compiled assembly does not return at the end, then add a return
		assembly += "\n\\"
	}
	state.compiledFunctions[functionName] = compiledFunction{assembly: assembly}

	// Return
	return codeParsingError{}
}

func (state *compilerState) compileFunctionCall(
	functionCall AbstractSyntaxTreeItem,
	parentFunctionArgs []functionArg,
	siblingFunctions map[string][]AbstractSyntaxTreeItem,
) (string, codeParsingError) {
	// TODO: Add support for functions having any as a register
	assert(functionCall.itemType == FunctionCall)

	// Check that the function is defined
	if _, functionDefined := siblingFunctions[functionCall.name]; !functionDefined {
		return "", codeParsingError{
			location: functionCall.location,
			msg:      errors.New("Call to undefined function `" + functionCall.name + "`"),
		}
	}

	// Parse function arguments
	functionDefinitionArgs := siblingFunctions[functionCall.name][0].contents
	if len(functionCall.contents) != len(functionDefinitionArgs) {
		return "", codeParsingError{
			location: functionCall.location,
			msg: errors.New(
				"`" + functionCall.name + "` takes " +
					fmt.Sprint(len(functionDefinitionArgs)) + " arguments, got " +
					fmt.Sprint(len(functionCall.contents)) + " arguments"),
		}
	}
	for index := range functionCall.contents {
		functionCallArg := functionCall.contents[index]
		functionDefintionArg := functionDefinitionArgs[index]

		// Check that the mutatability is equal
		functionCallArgMutatability, err := parseArgMutatability(functionCallArg.contents[0])
		if err.msg != nil {
			return "", err
		}
		functionDefintionArgMutatability, err := parseArgMutatability(functionDefintionArg.contents[0])
		if err.msg != nil {
			return "", err
		}
		if functionCallArgMutatability != functionDefintionArgMutatability {
			return "", codeParsingError{
				location: functionCallArg.contents[0].location,
				msg:      errors.New("`" + functionCall.name + "` takes " + functionDefintionArg.contents[0].name + " as mutatability, got " + functionCallArg.contents[0].name),
			}
		}

		// Check a constant variable is not being passed as a mutatable function arg
		if functionCallArgMutatability != Arg {
			for _, functionArg := range parentFunctionArgs {
				if functionArg.codeName == functionCallArg.name && functionArg.mutatability == Arg {
					return "", codeParsingError{
						location: functionCallArg.location,
						msg:      errors.New("Constant variable of mutatability " + functionArg.mutatability.String() + " cannot be passed as mutable variable of mutatability " + functionCallArgMutatability.String()),
					}
				}
			}
		}

		// Check that the register is equal
		assert(functionDefintionArg.contents[1].itemType == ArgRegister)
		functionDefinitionArgRegister := "%" + functionDefintionArg.contents[1].name
		functionCallArgRegister, err := state.getAssemblyRegisterFromVariableName(parentFunctionArgs, functionCallArg.name, functionCallArg.location)
		if err.msg != nil {
			return "", err
		}
		if functionDefinitionArgRegister != functionCallArgRegister {
			return "", codeParsingError{
				location: functionCallArg.location,
				msg:      errors.New("`" + functionCall.name + "` expects this argument to use the " + functionDefinitionArgRegister + " register, got " + functionCallArgRegister + " as register"),
			}
		}
	}

	// Compile the function if it has not been compiled already
	if _, functionExists := state.compiledFunctions[functionCall.name]; !functionExists {
		err := state.compileFunctionDefinition(siblingFunctions[functionCall.name], functionCall.name, siblingFunctions)
		if err.msg != nil {
			return "", err
		}
	}

	// Increase the references to the function
	entry, functionExists := state.compiledFunctions[functionCall.name]
	assert(functionExists)
	entry.references++
	state.compiledFunctions[functionCall.name] = entry

	// Return
	return "\n/" + functionCall.name + "/", codeParsingError{}
}

func compileAssembly(AST []AbstractSyntaxTreeItem) (string, codeParsingError) {
	assert(len(AST) != 0)

	// Get all of the globally declared functions in the AST
	globalFunctions := make(map[string][]AbstractSyntaxTreeItem)
	for _, ASTitem := range AST {
		if ASTitem.itemType == FunctionDefintion {
			if globalFunctions[ASTitem.name] != nil {
				return "", codeParsingError{
					location: ASTitem.location,
					msg: errors.New(
						"Second declaration of a function called `" + ASTitem.name +
							"`. Fuctions can only be declared once. The first declaration is at: " +
							fmt.Sprint(globalFunctions[ASTitem.name][0].location.line) +
							", " + fmt.Sprint(globalFunctions[ASTitem.name][0].location.column),
					),
				}
			}
			globalFunctions[ASTitem.name] = ASTitem.contents
		} else {
			// TODO: Handle AST items other then functions
		}
	}

	// Check that the main function exists
	if globalFunctions["main"] == nil {
		return "", codeParsingError{
			location: textLocation{
				line:   1,
				column: 1,
			},
			msg: errors.New("Could not find main function defintion"),
		}
	}

	// Compile the main function into assembly that has `\` to return from
	// functions, and `/FUNCTION_NAME/` to call other functions.
	state := compilerState{compiledFunctions: make(map[string]compiledFunction)}
	err := state.compileFunctionDefinition(globalFunctions["main"], "main", globalFunctions)
	if err.msg != nil {
		return "", err
	}

	// Compile the `\` to return from functions, and `/FUNCTION_NAME/` to call
	// other functions into valid assembly.
	// TODO: Change the return code for platforms other then linux X86-64
	state.transformFunctionDefinitionIntoValidAssembly("main", "mov $60, %rax\nmov $0, %rdi\nsyscall")

	// Concatanate the output
	out := ".global " + state.compiledFunctions["main"].jumpLabel + "\n.text" + state.dataSection
	for _, function := range state.compiledFunctions {
		out += function.assembly
	}
	return out + "\n", codeParsingError{}
}

func (state *compilerState) transformFunctionDefinitionIntoValidAssembly(functionName string, returnAssembly string) {
	functionDefinition, ok := state.compiledFunctions[functionName]
	assert(ok == true)
	if functionDefinition.jumpLabel != "" {
		return
	}

	// Update the function jump label so that when we call
	// `getAssemblyForFunctionCall` if it calls this function, then the early
	// return above can return before this function calls
	// `getAssemblyForFunctionCall` again and possibly start an infinite loop.
	if functionName == "main" {
		functionDefinition.jumpLabel = "_start"
	} else {
		functionDefinition.jumpLabel = state.createNewJumpLabel()
	}
	state.compiledFunctions[functionName] = functionDefinition

	// Change `functionDefinition.assembly` so that it is valid assembly
	functionDefinition.assembly = "\n" + functionDefinition.jumpLabel + ":" + strings.Replace(functionDefinition.assembly, "\\", returnAssembly, -1)
	for index := 0; index < len(functionDefinition.assembly); index++ {
		if functionDefinition.assembly[index] == '/' {
			index++
			assert(index < len(functionDefinition.assembly))
			functionToCall := ""
			for functionDefinition.assembly[index] != '/' {
				functionToCall += string(functionDefinition.assembly[index])
				index++
				assert(index < len(functionDefinition.assembly))
			}
			functionDefinition.assembly = strings.Replace(functionDefinition.assembly, "/"+functionToCall+"/", state.getAssemblyForFunctionCall(functionToCall), -1)
		}
	}
	state.compiledFunctions[functionName] = functionDefinition
}

func (state *compilerState) getAssemblyForFunctionCall(functionName string) string {
	if state.compiledFunctions[functionName].references <= 0 {
		panic("In `getAssemblyForFunctionCall`, function references expected to be greater then 0")
	} else if state.compiledFunctions[functionName].references == 1 {
		callerJumpLabel := state.createNewJumpLabel()
		state.transformFunctionDefinitionIntoValidAssembly(functionName, "jmp "+callerJumpLabel)
		return "jmp " + state.compiledFunctions[functionName].jumpLabel + "\n" + callerJumpLabel + ":"
	} else {
		state.transformFunctionDefinitionIntoValidAssembly(functionName, "ret")
		return "call " + state.compiledFunctions[functionName].jumpLabel
	}
}

func (state *compilerState) getAssemblyRegisterFromVariableName(parentFunctionArgs []functionArg, variableName string, variableLocation textLocation) (string, codeParsingError) {
	for _, functionArg := range parentFunctionArgs {
		if functionArg.codeName == variableName {
			return "%" + functionArg.register, codeParsingError{}
		}
	}
	return "", codeParsingError{
		location: variableLocation,
		msg:      errors.New("Could not find a variable called " + variableName),
	}
}

func (state *compilerState) convertValueToAssembly(parentFunctionArgs []functionArg, value AbstractSyntaxTreeItem) (string, codeParsingError) {
	switch value.itemType {
	// TODO: Add support for floats
	case Int:
		return "$" + value.name, codeParsingError{}
	case VarName:
		return state.getAssemblyRegisterFromVariableName(parentFunctionArgs, value.name, value.location)
	case String:
		dataSectionLabelForString := state.createNewDataSectionLabel()
		state.dataSection += "\n" + dataSectionLabelForString + ": .ascii \"" + value.name + "\""
		return "$" + dataSectionLabelForString, codeParsingError{}
	default:
		return "", codeParsingError{
			location: value.location,
			msg:      errors.New("Cannot parse value of type " + value.itemType.String()),
		}
	}
}

func (state *compilerState) conditionToAssembly(parentFunctionArgs []functionArg, condition AbstractSyntaxTreeItem, jumpToOnTrue string, jumpToOnFalse string) (string, codeParsingError) {
	assert(condition.itemType == ValueComparison || condition.itemType == BooleanLogic)
	assert(jumpToOnTrue != "" || jumpToOnFalse != "")
	out := ""
	switch condition.name {
	case "and", "or":
		for i, clause := range condition.contents {
			var assembly string
			var err codeParsingError
			if i < len(condition.contents)-1 {
				if condition.name == "and" {
					assembly, err = state.conditionToAssembly(parentFunctionArgs, clause, "", jumpToOnFalse)
				} else if condition.name == "or" {
					assembly, err = state.conditionToAssembly(parentFunctionArgs, clause, jumpToOnTrue, "")
				} else {
					assert(false)
				}
			} else {
				assembly, err = state.conditionToAssembly(parentFunctionArgs, clause, jumpToOnTrue, jumpToOnFalse)
			}
			if err.msg != nil {
				return "", err
			}
			out += assembly
		}
	case "<", ">=", ">", "<=", "==", "!=":
		assert(len(condition.contents) == 2)
		firstArg, err := state.convertValueToAssembly(parentFunctionArgs, condition.contents[0])
		if err.msg != nil {
			return "", err
		}
		secondArg, err := state.convertValueToAssembly(parentFunctionArgs, condition.contents[1])
		if err.msg != nil {
			return "", err
		}
		out += "\ncmp " + firstArg + ", " + secondArg

		var jumpOnTrueCmp, jumpOnFalseCmp string
		switch condition.name {
		case ">":
			jumpOnTrueCmp = "jl"
			jumpOnFalseCmp = "jge"
		case ">=":
			jumpOnTrueCmp = "jle"
			jumpOnFalseCmp = "jg"
		case "<":
			jumpOnTrueCmp = "jg"
			jumpOnFalseCmp = "jle"
		case "<=":
			jumpOnTrueCmp = "jge"
			jumpOnFalseCmp = "jl"
		case "==":
			jumpOnTrueCmp = "je"
			jumpOnFalseCmp = "jne"
		case "!=":
			jumpOnTrueCmp = "jne"
			jumpOnFalseCmp = "je"
		}

		if jumpToOnTrue != "" {
			out += "\n" + jumpOnTrueCmp + " " + jumpToOnTrue
			if jumpToOnFalse != "" {
				out += "\njmp " + jumpToOnFalse
			}
		} else if jumpToOnFalse != "" {
			out += "\n" + jumpOnFalseCmp + " " + jumpToOnFalse
		}
	default:
		assert(false)
	}
	return out, codeParsingError{}
}
