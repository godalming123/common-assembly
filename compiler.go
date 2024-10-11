package main

import (
	"errors"
	"fmt"
)

// Compiler.go
// ===========
// Responsible for compiling an abstract syntax tree into assembly

type compilerState struct {
	numberOfJumps              uint
	numberOfItemsInDataSection uint
	dataSection                string
}

func (jumpState *compilerState) createNewJumpLabel() string {
	jumpState.numberOfJumps++
	return "jumpLabel" + fmt.Sprint(jumpState.numberOfJumps)
}

func (jumpState *compilerState) createNewDataSectionLabel() string {
	jumpState.numberOfItemsInDataSection++
	return "dataSectionLabel" + fmt.Sprint(jumpState.numberOfItemsInDataSection)
}

type mutatability uint8

type functionDefinition struct {
	name string
	args []functionArg
}

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

type assemblyForControlFlowKeywords struct {
	continueAssembly string
	breakAssembly    string
	returnAssembly   string
}

func (state *compilerState) compileBlockToAssembly(
	parentFunction functionDefinition,
	block []AbstractSyntaxTreeItem,
	controlFlowKeywordsAssembly assemblyForControlFlowKeywords,
) (string, codeParsingError) {
	assembly := ""
loop:
	for index, statement := range block {
		// TODO: Support function calls, and function definitions in funtions
		switch statement.itemType {

		case Return:
			assert(index == len(block)-1)
			assembly += controlFlowKeywordsAssembly.returnAssembly
			break loop

		case AssemblySyscall:
			assembly += "\nsyscall"

		case VariableMutation:
			assert(len(statement.contents) == 2)
			assembly += "\n" + statement.name + " "
			firstArg, err := state.convertValueToAssembly(parentFunction, statement.contents[0])
			if err.msg != nil {
				return "", err
			}
			secondArg, err := state.convertValueToAssembly(parentFunction, statement.contents[1])
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
			loopBodyAssembly, err := state.compileBlockToAssembly(parentFunction, statement.contents[1:], assemblyForControlFlowKeywords{
				returnAssembly:   controlFlowKeywordsAssembly.returnAssembly,
				breakAssembly:    "\njmp" + loopEndJumpLabel,
				continueAssembly: "\njmp" + loopConditionJumpLabel,
			})
			if err.msg != nil {
				return "", err
			}
			assembly += loopBodyAssembly

			// Add loop condition
			assembly += "\n" + loopConditionJumpLabel + ":"
			conditionAssembly, err := state.conditionToAssembly(parentFunction, statement.contents[0], loopBodyJumpLabel, "")
			if err.msg != nil {
				return "", err
			}
			assembly += conditionAssembly

			// Add loop end
			assembly += "\n" + loopEndJumpLabel + ":"

		case IfStatement:
			elseBlockJumpLabel := state.createNewJumpLabel()
			ifCheck, err := state.conditionToAssembly(parentFunction, statement.contents[0], "", elseBlockJumpLabel)
			if err.msg != nil {
				return "", err
			}
			assembly += ifCheck
			if statement.contents[len(statement.contents)-1].itemType == ElseStatement {
				ifBody, err := state.compileBlockToAssembly(parentFunction, statement.contents[1:len(statement.contents)-1], controlFlowKeywordsAssembly)
				if err.msg != nil {
					return "", err
				}
				assembly += ifBody
				endJumpLabel := state.createNewJumpLabel()
				assembly += "\njmp " + endJumpLabel
				elseBody, err := state.compileBlockToAssembly(parentFunction, statement.contents[len(statement.contents)-1].contents, controlFlowKeywordsAssembly)
				if err.msg != nil {
					return "", err
				}
				assembly += "\n" + elseBlockJumpLabel + ":"
				assembly += elseBody
				assembly += "\n" + endJumpLabel + ":"
			} else {
				ifBody, err := state.compileBlockToAssembly(parentFunction, statement.contents[1:], controlFlowKeywordsAssembly)
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

// Compiles a function from an AST item into X86-64 assmebly which is output as
// a string.
func (state *compilerState) compileFunctionToAssembly(
	functionAST AbstractSyntaxTreeItem,
	functionArgumentsAreValid func([]functionArg) (int, error),
	controlFlowKeywordsAssembly assemblyForControlFlowKeywords,
) (string, codeParsingError) {
	assert(functionAST.itemType == FunctionDefintion)

	// Get the function args
	assert(functionAST.contents[0].itemType == FunctionArgs)
	funcArgs := []functionArg{}
	for _, funcArg := range functionAST.contents[0].contents {
		assert(funcArg.contents[0].itemType == ArgMutatability)
		if funcArg.contents[0].name != "mut" {
			return "", codeParsingError{
				msg: errors.New("Argemunts for the main function can only be `mut`."),
			}
		}
		assert(funcArg.contents[1].itemType == ArgRegister)
		add(&funcArgs, functionArg{
			codeName:     funcArg.name,
			mutatability: Mut,
			register:     funcArg.contents[1].name,
		})
	}

	// Check that the function args are valid
	errArgIndex, err := functionArgumentsAreValid(funcArgs)
	if err != nil {
		return "", codeParsingError{
			location: functionAST.contents[0].contents[errArgIndex].location,
			msg:      err,
		}
	}

	// Return
	return state.compileBlockToAssembly(
		functionDefinition{
			name: functionAST.name,
			args: funcArgs,
		},
		functionAST.contents[1:],
		controlFlowKeywordsAssembly,
	)
}

func compileAssembly(AST []AbstractSyntaxTreeItem) (string, string, codeParsingError) {
	assert(len(AST) != 0)

	// Find the index into `AST` of the main function
	mainFuncIndex := 0
	for AST[mainFuncIndex].itemType != FunctionDefintion ||
		AST[mainFuncIndex].name != "main" {
		mainFuncIndex++
		if mainFuncIndex >= len(AST) {
			return "", "", codeParsingError{
				location: textLocation{
					line:   1,
					column: 1,
				},
				msg: errors.New("Could not find a main function"),
			}
		}
	}

	// Compile the main function
	state := compilerState{}
	assembly, err := state.compileFunctionToAssembly(
		AST[mainFuncIndex],
		func(args []functionArg) (int, error) {
			for index, arg := range args {
				if arg.mutatability != Mut {
					return index, errors.New("Argemunts for the main function can only be `mut`.")
				}
			}
			return 0, nil
		},
		// TODO: Change the return code for platforms other then linux X86-64
		assemblyForControlFlowKeywords{
			returnAssembly: "\nmov $60, %rax\nmov $0, %rdi\nsyscall",
		},
	)

	// Return the result
	return state.dataSection, "\n_start:" + assembly, err
}

func (state *compilerState) getAssemblyNameFromVariableName(parentFunction functionDefinition, variable AbstractSyntaxTreeItem) (string, codeParsingError) {
	assert(variable.itemType == VarName)
	for _, functionArg := range parentFunction.args {
		if functionArg.codeName == variable.name {
			return "%" + functionArg.register, codeParsingError{}
		}
	}
	return "", codeParsingError{
		location: variable.location,
		msg:      errors.New("Could not find a variable called " + variable.name),
	}
}

func (state *compilerState) convertValueToAssembly(parentFunction functionDefinition, value AbstractSyntaxTreeItem) (string, codeParsingError) {
	switch value.itemType {
	// TODO: Add support for strings and floats
	case Int:
		return "$" + value.name, codeParsingError{}
	case VarName:
		return state.getAssemblyNameFromVariableName(parentFunction, value)
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

func (state *compilerState) conditionToAssembly(parentFunction functionDefinition, condition AbstractSyntaxTreeItem, jumpToOnTrue string, jumpToOnFalse string) (string, codeParsingError) {
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
					assembly, err = state.conditionToAssembly(parentFunction, clause, "", jumpToOnFalse)
				} else if condition.name == "or" {
					assembly, err = state.conditionToAssembly(parentFunction, clause, jumpToOnTrue, "")
				} else {
					assert(false)
				}
			} else {
				assembly, err = state.conditionToAssembly(parentFunction, clause, jumpToOnTrue, jumpToOnFalse)
			}
			if err.msg != nil {
				return "", err
			}
			out += assembly
		}
	case "<", ">=", ">", "<=", "==", "!=":
		assert(len(condition.contents) == 2)
		firstArg, err := state.convertValueToAssembly(parentFunction, condition.contents[0])
		if err.msg != nil {
			return "", err
		}
		secondArg, err := state.convertValueToAssembly(parentFunction, condition.contents[1])
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
