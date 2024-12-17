package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Compiler.go
// ===========
// Responsible for compiling an abstract syntax tree into assembly

type idividualRegisterState struct {
	// If the variableName == "", then this register is not assigned to a variable
	variableName string

	// Stores if a variable can be dropped in the current scope, for example if you define a variable
	// outside a while loop, then it cannot be dropped inside the while loop.
	stopVariableFromBeingDropped bool

	// A variable name can be defined at a different location to where the register was defined as
	// mutatable, for example:
	// fn b0 = myFunc() {
	//  # ^ register was defined as mutatable here
	//     b0 name = "John"
	//      # ^ variable name was defined here
	// }

	// This location is saved so that the compiler can warn the user if a variable is not used
	variableNameWasDefinedAt textLocation

	// This location is saved so that the compiler can warn the user if a register that was defined as
	// mutatable is not mutated. If registerWasDefinedAsMutatableAt == textLocation{}, then this
	// register was not defined as mutatable.
	registerWasDefinedAsMutatableAt textLocation
}

type registerState struct {
	registers [16]idividualRegisterState

	// The registers that the surrounding function uses to return values to the caller. This is a
	// subset of the registers that the surroinding function can mutate.
	functionReturnValueRegisters []int
}

func commonAssemblyRegisterToX86Register(registerIndex int) string {
	switch registerIndex {
	case 0:
		return "%rax"
	case 1:
		return "%rbx"
	case 2:
		return "%rcx"
	case 3:
		return "%rdx"
	case 4:
		return "%rsi"
	case 5:
		return "%rdi"
	case 6:
		return "%r8"
	case 7:
		return "%r9"
	case 8:
		return "%r10"
	case 9:
		return "%r11"
	case 10:
		return "%r12"
	case 11:
		return "%r13"
	case 12:
		return "%r14"
	case 13:
		return "%r15"
	case 14:
		return "%rsp"
	case 15:
		return "%ebp"
	default:
		panic("The number " + fmt.Sprint(registerIndex) + " does not correspond to an X86-64 register")
	}
}

type compiledFunction struct {
	references uint
	jumpLabel  string
	// If jumpLabel == "", then this code will have `\` to return from this
	// function, and maybe `/FUNCTION_NAME/` to call other functions. Therefore
	// this code might still need to be compiled to assembly.
	assembly string
}

type compilerState struct {
	numberOfJumps              uint
	numberOfItemsInDataSection uint
	dataSection                string
	compiledFunctions          map[string]compiledFunction
}

func (state *compilerState) createNewJumpLabel() string {
	state.numberOfJumps++
	return "jumpLabel" + fmt.Sprint(state.numberOfJumps)
}

func (state *compilerState) createNewDataSectionLabel() string {
	state.numberOfItemsInDataSection++
	return "dataSectionLabel" + fmt.Sprint(state.numberOfItemsInDataSection)
}

// Stores the assembly code to be inserted when a control flow keyword is used.
// If the assembly code is a blank string, then that control flow cannot be used
// in the current scope.
type assemblyForControlFlowKeywords struct {
	continueAssembly string
	breakAssembly    string
}

// Modifies the register states so that inner scope cannot drop variables defined in outer scope
func parseRegisterStatesToInnerScope(regState registerState) registerState {
	for i := range regState.registers {
		if regState.registers[i].variableName != "" {
			regState.registers[i].stopVariableFromBeingDropped = true
		}
	}
	return regState
}

func (state *compilerState) compileBlockToAssembly(
	block []ASTitem,
	regState registerState,
	siblingFunctions map[string][]ASTitem,
	controlFlowKeywordsAssembly assemblyForControlFlowKeywords,
) (string, []codeParsingError) {
	assembly := ""
	for index, statement := range block {
		switch statement.itemType {

		case FunctionReturn:
			assert(eq(index, len(block)-1))
			assemblyForArgs, returnRegisters, errs := state.compileFunctionCallArguments(statement.contents, &regState, false)
			if len(errs) != 0 {
				return "", errs
			}
			err := checkRegisterListsAreTheSame(regState.functionReturnValueRegisters, returnRegisters)
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			return assembly + assemblyForArgs + "\n\\", []codeParsingError{}

		case Increment, Decrement:
			assert(eq(len(statement.contents), 1))
			assert(eq(statement.contents[0].itemType, SetToAValue))
			assert(eq(len(statement.contents[0].contents), 1))
			assert(eq(statement.contents[0].contents[0].itemType, Register))
			assert(eq(len(statement.contents[0].contents[0].contents), 1))
			value, err := state.convertValueToAssembly(&regState, statement.contents[0].contents[0].contents[0])
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			if statement.itemType == Increment {
				assembly += "\ninc " + value
			} else {
				assembly += "\ndec " + value
			}

		case Assignment, PlusEquals, MinusEquals, MultiplyEquals, DivideEquals:
			assert(eq(len(statement.contents), 2))
			assert(eq(statement.contents[0].itemType, SetToAValue))
			assemblyForStatement := ""
			errs := []codeParsingError{}
			switch statement.contents[1].itemType {
			default:
				panic("Unexpected internal state: `statement.contents[1].itemType` equals " + statement.contents[1].itemType.String())
			case Function:
				assemblyForStatement, errs = state.compileFunctionCall(statement, &regState, siblingFunctions)
			case Name, IntNumber, FloatNumber, StringValue, BoolValue, CharValue, DropVariable:
				assemblyForStatement, errs = state.compileVariableMutation(statement, &regState)
			}
			if len(errs) != 0 {
				return "", errs
			}
			assembly += assemblyForStatement

		case WhileLoop:
			// Save jump labels
			loopBodyJumpLabel := state.createNewJumpLabel()
			loopConditionJumpLabel := state.createNewJumpLabel()
			loopEndJumpLabel := state.createNewJumpLabel()

			// Add loop head
			assembly += "\njmp " + loopConditionJumpLabel

			// Add loop body
			assembly += "\n" + loopBodyJumpLabel + ":"
			loopBodyAssembly, errs := state.compileBlockToAssembly(statement.contents[1:], parseRegisterStatesToInnerScope(regState), siblingFunctions, assemblyForControlFlowKeywords{
				breakAssembly:    "\njmp " + loopEndJumpLabel,
				continueAssembly: "\njmp " + loopConditionJumpLabel,
			})
			if len(errs) != 0 {
				return "", errs
			}
			assembly += loopBodyAssembly

			// Add loop condition
			assembly += "\n" + loopConditionJumpLabel + ":"
			conditionAssembly, err := state.conditionToAssembly(&regState, statement.contents[0], loopBodyJumpLabel, "")
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			assembly += conditionAssembly

			// Add loop end
			assembly += "\n" + loopEndJumpLabel + ":"

		case IfStatement:
			elseBlockJumpLabel := state.createNewJumpLabel()
			ifCheck, err := state.conditionToAssembly(&regState, statement.contents[0], "", elseBlockJumpLabel)
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			assembly += ifCheck
			innerScopeRegStates := parseRegisterStatesToInnerScope(regState)
			if statement.contents[len(statement.contents)-1].itemType == ElseStatement {
				ifBody, errs := state.compileBlockToAssembly(statement.contents[1:len(statement.contents)-1], innerScopeRegStates, siblingFunctions, controlFlowKeywordsAssembly)
				if len(errs) != 0 {
					return "", errs
				}
				assembly += ifBody
				endJumpLabel := state.createNewJumpLabel()
				assembly += "\njmp " + endJumpLabel
				elseBody, errs := state.compileBlockToAssembly(statement.contents[len(statement.contents)-1].contents, innerScopeRegStates, siblingFunctions, controlFlowKeywordsAssembly)
				if len(errs) != 0 {
					return "", errs
				}
				assembly += "\n" + elseBlockJumpLabel + ":"
				assembly += elseBody
				assembly += "\n" + endJumpLabel + ":"
			} else {
				ifBody, errs := state.compileBlockToAssembly(statement.contents[1:], innerScopeRegStates, siblingFunctions, controlFlowKeywordsAssembly)
				if len(errs) != 0 {
					return "", errs
				}
				assembly += ifBody
				assembly += "\n" + elseBlockJumpLabel + ":"
			}

		case BreakStatement:
			if controlFlowKeywordsAssembly.breakAssembly == "" {
				return "", []codeParsingError{{
					msg:      errors.New("Break statement is not valid in this scope"),
					location: statement.location,
				}}
			}
			assembly += controlFlowKeywordsAssembly.breakAssembly
		case ContinueStatement:
			if controlFlowKeywordsAssembly.continueAssembly == "" {
				return "", []codeParsingError{{
					msg:      errors.New("Continue statement is not valid in this scope"),
					location: statement.location,
				}}
			}
			assembly += controlFlowKeywordsAssembly.continueAssembly

		case DropVariable:
			_, err := getRegisterFromVariableASTitem(&regState, statement)
			if err.msg != nil {
				return "", []codeParsingError{err}
			}

		default:
			return "", []codeParsingError{{
				msg: errors.New("Expecting AST nodes in the body of a block to either be " +
					"of type Return, FunctionCall, AssemblySyscall, VariableMutation, " +
					"WhileLoop, IfStatement, BreakStatement, or ContinueStatement, got " +
					"an AST node of type `" +
					statement.itemType.String() + "`."),
				location: statement.location,
			}}
		}
	}
	return assembly, []codeParsingError{}
}

type registerAndLocation struct {
	register int
	location textLocation
}

// Compiles the arguments in a function call into assembly
func (state *compilerState) compileFunctionCallArguments(
	functionArguments []ASTitem,
	regState *registerState,

	// Wether to check for if a variable is mutated without naming the variable by naming the register
	// associated with the variable. This is turned off when this function is used to compile return
	// statements, since in that case it does not matter if a variable declared in the fuction is
	// implicitly mutated since the function is being returned from.
	checkImplicitVariableMutation bool,
) (
	string, // The assembly for the function arguments
	[]registerAndLocation, // The list of registers of the function arguments
	[]codeParsingError,
) {
	assembly := ""
	registers := []registerAndLocation{}
	for _, arg := range functionArguments {
		assert(eq(arg.itemType, Register))
		assert(eq(len(arg.contents), 1))

		argRegister := -1
		if arg.name == "" {
			var err codeParsingError
			argRegister, err = getRegisterFromVariableASTitem(regState, arg.contents[0])
			if err.msg != nil {
				return "", []registerAndLocation{}, []codeParsingError{err}
			}
		} else {
			argRegister = ASTitemToRegister(arg)
			assert(notEq(argRegister, -1))

			if regState.registers[argRegister].registerWasDefinedAsMutatableAt.line == 0 {
				return "", []registerAndLocation{}, []codeParsingError{{
					location: arg.location,
					msg:      errors.New("It is not possible to mutate the register r" + fmt.Sprint(argRegister) + "."),
				}}
			}

			if checkImplicitVariableMutation && regState.registers[argRegister].variableName != "" {
				return "", []registerAndLocation{}, []codeParsingError{{
					location: arg.location,
					msg:      errors.New("It is only possible to mutate the register r" + fmt.Sprint(argRegister) + " through the variable " + regState.registers[argRegister].variableName),
				}}
			}

			argValue, err := state.convertValueToAssembly(regState, arg.contents[0])
			if err.msg != nil {
				return "", []registerAndLocation{}, []codeParsingError{err}
			}

			assembly += "\nmov " + argValue + ", " + commonAssemblyRegisterToX86Register(argRegister)
		}

		for _, register := range registers {
			if register.register == argRegister {
				errMsg := errors.New("Register r" + fmt.Sprint(register) + " used atleast twice in function arguments. Each register can only be used once.")
				return "", []registerAndLocation{}, []codeParsingError{
					{msg: errMsg, location: register.location},
					{msg: errMsg, location: arg.location},
				}
			}
		}
		add(&registers, registerAndLocation{
			register: argRegister,
			location: arg.location,
		})
	}
	return assembly, registers, []codeParsingError{}
}

func ASTitemToRegister(ASTitem ASTitem) int {
	assert(eq(ASTitem.itemType, Register))
	if len(ASTitem.name) > 0 {
		assert(eq(ASTitem.name[0], 'r'))
		register, err := strconv.Atoi(ASTitem.name[1:])
		assert(eq(err, nil))
		return register
	}
	return -1
}

type mutatedRegister struct {
	register     int
	location     textLocation
	variableName string
	// The number of times to modify the value the register points to rather then the register itself
	pointerDereferenceLayers uint
}

func ASTitemToMutatedRegister(item ASTitem) mutatedRegister {
	register := ASTitemToRegister(item)
	variableName := ""
	var pointerDereferenceLayers uint = 0

	if len(item.contents) > 0 {
		assert(eq(len(item.contents), 1))
		variable := item.contents[0]
		for variable.itemType == Dereference {
			pointerDereferenceLayers++
			assert(eq(variable.name, ""))
			assert(eq(len(variable.contents), 1))
			variable = variable.contents[0]
		}
		assert(eq(variable.itemType, Name))
		variableName = variable.name
	}

	return mutatedRegister{
		register:                 register,
		variableName:             variableName,
		pointerDereferenceLayers: pointerDereferenceLayers,
		location:                 item.location,
	}
}

// Parses the values on the left side of the equals, and checks that they are valid including:
// - A register that is reserved for a variable is implicityly mutated without naming the variable
// - An undefined variable has been mutated
// - A defined variable has been re-defined
// - A register that the surrounding function does not mark as mutatable has been mutated
func parseAndValidateMutatedValueOnLeftSideOfEqualsInAssignment(item ASTitem, regState *registerState) (mutatedRegister, []codeParsingError) {
	// Initialise variables
	mutatedValue := ASTitemToMutatedRegister(item)
	registerTheVariableWasAlreadyDefinedToUse := -1
	errs := []codeParsingError{}

	// Handle the mutated register if there is one
	if mutatedValue.register != -1 {
		// Check if the register is already reserved for another variable
		if regState.registers[mutatedValue.register].variableName != "" {
			add(&errs, codeParsingError{
				msg: errors.New("The register r" + fmt.Sprint(mutatedValue.register) + " is already reserved for a " +
					"variable called `" + regState.registers[mutatedValue.register].variableName + "`. If you want to stop using the " +
					"old variable, then add `drop " + regState.registers[mutatedValue.register].variableName + "` before this line of" +
					" code. If you want to continue using the old variable, then you have 2 options. Your " +
					"first option is to refactor your code so that either this line of code, or line " +
					fmt.Sprint(regState.registers[mutatedValue.register].variableNameWasDefinedAt) + " where the `" +
					regState.registers[mutatedValue.register].variableName + "` variable was defined does not use the r" +
					fmt.Sprint(mutatedValue.register) + " register. Your second option is to copy the old variable to a " +
					"different register."),
				location: item.location,
			})
		}
	}

	// Handle the mutated variable if there is one
	if mutatedValue.variableName != "" {
		// Get the register that was already defined to use this variable if there is one
		for variableRegister, variable := range regState.registers {
			if variable.variableName == mutatedValue.variableName {
				registerTheVariableWasAlreadyDefinedToUse = variableRegister
				break
			}
		}

		// Check possible errors
		if registerTheVariableWasAlreadyDefinedToUse == -1 {
			if mutatedValue.register == -1 {
				// The user has tried to mutate a variable that has not been defined yet
				add(&errs, codeParsingError{
					msg: errors.New("You have tried to mutate a variable (`" + mutatedValue.variableName + "`) that has not " +
						"been defined yet. If you want to define this variable, then add the register that this" +
						" variable will use next to the variable name."),
					location: item.location,
				})
			}
		} else if mutatedValue.register != -1 {
			// The user has tried to re-define a variable that is already defined
			if registerTheVariableWasAlreadyDefinedToUse != mutatedValue.register {
				add(&errs, codeParsingError{
					msg: errors.New("`" + mutatedValue.variableName + "` is already defined as using the register r" +
						fmt.Sprint(registerTheVariableWasAlreadyDefinedToUse) + ", however here you are trying to " +
						" redefine this variable to use a different register (r" + fmt.Sprint(mutatedValue.register) + "). If " +
						"you want to stop using the old variable, then add `drop" + mutatedValue.variableName + "` before this " +
						"line of code. If you want to use both variables, then you will have to change the name of " +
						"one of the variables."),
					location: item.location,
				})
			} else {
				add(&errs, codeParsingError{
					msg: errors.New("Variable ( " + mutatedValue.variableName + ") and register (r" + fmt.Sprint(mutatedValue.register) +
						") named to mutate a variable that is already defined. After a variable has been " +
						"defined, it can be mutated by just naming the variable instead of naming the variable " +
						"and the register."),
					location: item.location,
				})
			}
		}
	}

	// Get the register the user mutated
	register := -1
	if mutatedValue.register == -1 {
		assert(notEq(registerTheVariableWasAlreadyDefinedToUse, -1))
		register = registerTheVariableWasAlreadyDefinedToUse
	} else {
		assert(eq(registerTheVariableWasAlreadyDefinedToUse, -1))
		register = mutatedValue.register
	}

	// Handle if the user tried to mutate a register that the surrounding function does not explicitly
	// mutate
	if regState.registers[register].registerWasDefinedAsMutatableAt.line == 0 {
		add(&errs, codeParsingError{
			msg: errors.New("You cannot mutate the r" + fmt.Sprint(register) + " register unless you add " +
				"it to the list of registers that the function mutates."),
			location: item.location,
		})
	}

	// Return if there are errors
	if len(errs) != 0 {
		return mutatedRegister{}, errs
	}

	// Handle updating register states
	if mutatedValue.register != -1 && mutatedValue.variableName != "" {
		assert(eq(regState.registers[register].variableName, ""))
		assert(eq(regState.registers[register].variableNameWasDefinedAt, textLocation{}))
		regState.registers[register].variableName = mutatedValue.variableName
		regState.registers[register].variableNameWasDefinedAt = item.location
	}

	// Return
	return mutatedRegister{
		register:                 register,
		location:                 item.location,
		variableName:             mutatedValue.variableName,
		pointerDereferenceLayers: mutatedValue.pointerDereferenceLayers,
	}, []codeParsingError{}
}

// Compiles a variableMutation ASTitem of type Assigment, PlusEquals, MinusEquals, MultiplyEquals or DivideEquals into assembly
func (state *compilerState) compileVariableMutation(
	statement ASTitem,
	regState *registerState,
) (string, []codeParsingError) {
	// Check that only one register is being mutated
	if len(statement.contents[0].contents) != 1 {
		return "", []codeParsingError{{
			location: statement.contents[0].location,
			msg: errors.New(
				"Expect 1 value on left side of equals unless a function is being called. Got " +
					fmt.Sprint(len(statement.contents[0].contents)) +
					" values on left side of equals being set to one value of type " +
					statement.contents[1].itemType.String() + ". Please split this into multiply assignments.",
			),
		}}
	}

	// Get the common assembly register that is being mutated, and update the register states
	mutatedRegister, errs := parseAndValidateMutatedValueOnLeftSideOfEqualsInAssignment(statement.contents[0].contents[0], regState)
	if len(errs) != 0 {
		return "", errs
	}

	// Check that the register is reserved for a variable
	if mutatedRegister.variableName == "" {
		return "", []codeParsingError{{
			location: mutatedRegister.location,
			msg: errors.New("Without giving the register a variable name, the value that you assign to it " +
				"here cannot be used later, so there is no point in assigning a value to a register without " +
				"reserving the register for use with a specefic variable."),
		}}
	}

	// Convert the common assembly register number into an x86 register
	mutatedRegisterAssembly := strings.Repeat("(", int(mutatedRegister.pointerDereferenceLayers)) +
		commonAssemblyRegisterToX86Register(mutatedRegister.register) +
		strings.Repeat(")", int(mutatedRegister.pointerDereferenceLayers))

	// Get the assembly for the value being assigned to the register
	valueBeingAssignedToVariable, err := state.convertValueToAssembly(regState, statement.contents[1])
	if err.msg != nil {
		return "", []codeParsingError{err}
	}

	// Get the assembly instruction to use
	instruction := ""
	switch statement.itemType {
	case Assignment:
		instruction = "mov"
	case PlusEquals:
		instruction = "add"
	case MinusEquals:
		instruction = "sub"
	case MultiplyEquals:
		instruction = "mul"
	case DivideEquals:
		instruction = "div"
	default:
		panic("Unexpected internal state: Expected statement.contents[1].itemType to equal either Function, StringValue, Name, or IntNumber")
	}

	// Return
	return "\n" + instruction + " " + valueBeingAssignedToVariable + ", " + mutatedRegisterAssembly, []codeParsingError{}
}

// Compiles a functionCall ASTitem of type Assigment, PlusEquals, MinusEquals, MultiplyEquals or DivideEquals into assembly
func (state *compilerState) compileFunctionCall(
	statement ASTitem,
	regState *registerState,
	siblingFunctions map[string][]ASTitem,
) (string, []codeParsingError) {
	// TODO: Add support for functions having any as a register
	functionName := statement.contents[1].name
	assert(notEq(functionName, ""))

	// Check that the function is defined, and get the code to call the function
	functionCallCode := ""
	_, isUserDefinedFunction := siblingFunctions[functionName]
	if isUserDefinedFunction {
		// Compile the function if it has not been compiled already
		errs := state.compileFunctionDefinition(siblingFunctions[statement.contents[1].name], siblingFunctions)
		if len(errs) != 0 {
			return "", errs
		}

		// Increase the references to the function
		entry, _ := state.compiledFunctions[functionName]
		entry.references++
		state.compiledFunctions[functionName] = entry

		// Set functionCallCode
		functionCallCode = "/" + functionName + "/"
	} else {
		switch functionName {
		case "sysRead":
			functionCallCode = "mov $0, %rax\nsyscall"
		case "sysWrite":
			functionCallCode = "mov $1, %rax\nsyscall"
		case "sysOpen":
			functionCallCode = "mov $2, %rax\nsyscall"
		case "sysClose":
			functionCallCode = "mov $3, %rax\nsyscall"
		case "sysBrk":
			functionCallCode = "mov $12, %rax\nsyscall"
		case "sysExit":
			functionCallCode = "mov $60, %rax\nsyscall"
		default:
			return "", []codeParsingError{{
				location: statement.contents[1].location,
				msg:      errors.New("Call to undefined function `" + functionName + "`"),
			}}
		}
	}

	// Compile the function arguments
	assemblyForArgs, functionCallArgRegisters, errs := state.compileFunctionCallArguments(statement.contents[1].contents, regState, true)
	if len(errs) != 0 {
		return "", errs
	}

	// Get the expected registers of the function arguments
	functionExpectedArgRegisters := []int{}
	if isUserDefinedFunction {
		functionExpectedArgRegisters = mapList(siblingFunctions[functionName][1].contents, ASTitemToRegister)
		for _, expectedArgRegister := range functionExpectedArgRegisters {
			assert(notEq(expectedArgRegister, -1))
		}
	} else {
		switch functionName {
		case "sysRead", "sysWrite", "sysOpen":
			functionExpectedArgRegisters = []int{5, 4, 3}
		case "sysClose", "sysBrk", "sysExit":
			functionExpectedArgRegisters = []int{5}
		default:
			panic("Unexpected internal state: isUserDefinedFunction is false, and functionName is `" + functionName + "`.")
		}
	}

	// Check that the function arguments use the expected registers
	err := checkRegisterListsAreTheSame(functionExpectedArgRegisters, functionCallArgRegisters)
	if err.msg != nil {
		return "", []codeParsingError{err}
	}

	// Get mutated registers
	functionMutatedRegisters := make([]mutatedRegister, len(statement.contents[0].contents))
	for index, item := range statement.contents[0].contents {
		functionMutatedRegisters[index], errs = parseAndValidateMutatedValueOnLeftSideOfEqualsInAssignment(item, regState)
		if len(errs) != 0 {
			return "", errs
		}
		if functionMutatedRegisters[index].pointerDereferenceLayers > 0 {
			return "", []codeParsingError{{
				msg:      errors.New("Mutated value in function call cannot be dereferenced with ^"),
				location: item.location,
			}}
		}
	}

	// Get the expected mutated registers
	functionExpectedMutatedRegisters := []mutatedRegister{}
	if isUserDefinedFunction {
		functionExpectedMutatedRegisters = mapList(siblingFunctions[functionName][0].contents, ASTitemToMutatedRegister)
		for _, expectedMutatedRegister := range functionExpectedMutatedRegisters {
			assert(eq(expectedMutatedRegister.pointerDereferenceLayers, 0))
		}
	} else {
		switch functionName {
		case "sysRead", "sysWrite", "sysClose", "sysBrk", "sysExit":
			functionExpectedMutatedRegisters = []mutatedRegister{{
				register:                 0,
				variableName:             "exitCode",
				pointerDereferenceLayers: 0,
			}}
		case "sysOpen":
			functionExpectedMutatedRegisters = []mutatedRegister{{
				register:                 0,
				variableName:             "fileDescriptor",
				pointerDereferenceLayers: 0,
			}}
		default:
			panic("Unexpected internal state: isUserDefinedFunction is false, and functionName is `" + functionName + "`.")
		}
	}

	// Check that the function mutated regisers use the expected registers
	err = checkRegisterListsAreTheSame(
		mapList(functionExpectedMutatedRegisters, func(in mutatedRegister) int {
			return in.register
		}),
		mapList(functionMutatedRegisters, func(in mutatedRegister) registerAndLocation {
			return registerAndLocation{
				register: in.register,
				location: in.location,
			}
		}),
	)
	if err.msg != nil {
		return "", []codeParsingError{err}
	}
	for i, expectedMutatedRegister := range functionExpectedMutatedRegisters {
		if expectedMutatedRegister.variableName == "" && functionMutatedRegisters[i].variableName != "" {
			return "", []codeParsingError{{
				location: functionMutatedRegisters[i].location,
				msg: errors.New("Function call stores the final value that the r" +
					fmt.Sprint(functionMutatedRegisters[i].register) + " register was mutated to in a new " +
					"variable called `" + functionMutatedRegisters[i].variableName + "`, but the function " +
					"definition does not garuntee that it will muatate that register."),
			}}
		}
	}

	// Return
	return assemblyForArgs + "\n" + functionCallCode, []codeParsingError{}
}

func parseFunctionDefinitionRegisters(mutatedRegisters []ASTitem, functionArgs []ASTitem) (registerState, []codeParsingError) {
	out := registerState{}

	// Parse the function mutated registers
	for _, mutatedRegister := range mutatedRegisters {
		// TODO: Check that the mutated registers do not have the same name
		register := ASTitemToMutatedRegister(mutatedRegister)
		assert(eq(register.pointerDereferenceLayers, 0))
		assert(notEq(register.register, -1))
		if register.variableName != "" {
			add(&out.functionReturnValueRegisters, register.register)
		}
		if out.registers[register.register].registerWasDefinedAsMutatableAt.line != 0 {
			errMsg := errors.New("Register " + mutatedRegister.name + " used twice in mutated registers")
			return registerState{}, []codeParsingError{
				{msg: errMsg, location: out.registers[register.register].registerWasDefinedAsMutatableAt},
				{msg: errMsg, location: mutatedRegister.location},
			}
		}
		out.registers[register.register].registerWasDefinedAsMutatableAt = mutatedRegister.location
	}

	// Parse the function args
	for _, arg := range functionArgs {
		mutatedRegister := ASTitemToMutatedRegister(arg)
		assert(notEq(mutatedRegister.register, -1))
		assert(notEq(mutatedRegister.variableName, ""))
		assert(eq(mutatedRegister.pointerDereferenceLayers, 0))

		// Check that the same register has not been used already
		if out.registers[mutatedRegister.register].variableName != "" {
			errMsg := errors.New("Register " + arg.name + " used twice in function arguments. Each register can only be used once.")
			return registerState{}, []codeParsingError{
				{msg: errMsg, location: out.registers[mutatedRegister.register].variableNameWasDefinedAt},
				{msg: errMsg, location: arg.location},
			}
		}

		// Check that the same name has not been used already
		for _, regState := range out.registers {
			if regState.variableName == mutatedRegister.variableName {
				errMsg := errors.New("Variable name " + mutatedRegister.variableName + " used twice in function arguments. Each variable name can only be used once.")
				return registerState{}, []codeParsingError{
					{msg: errMsg, location: out.registers[mutatedRegister.register].variableNameWasDefinedAt},
					{msg: errMsg, location: arg.location},
				}
			}
		}

		// Update registerStates
		out.registers[mutatedRegister.register].variableName = mutatedRegister.variableName
		out.registers[mutatedRegister.register].variableNameWasDefinedAt = arg.contents[0].location
	}

	// Return
	return out, []codeParsingError{}
}

func (state *compilerState) compileFunctionDefinition(
	functionDefinition []ASTitem,
	siblingFunctions map[string][]ASTitem,
) []codeParsingError {
	assert(eq(functionDefinition[1].itemType, Function))
	functionName := functionDefinition[1].name
	assert(notEq(functionName, ""))

	// Add the `functionName` key to the compiledFunctions hashmap so that when
	// `compileBlockToAssembly()` calls `compileFunctionCall`, that does not call
	// this function if the function being called is the current function being
	// compiled to stop an infinite loop.
	state.compiledFunctions[functionName] = compiledFunction{}

	// Parse registers that the function mutates
	assert(eq(functionDefinition[1].itemType, Function))
	assert(eq(functionDefinition[0].itemType, SetToAValue))
	regState, errs := parseFunctionDefinitionRegisters(functionDefinition[0].contents, functionDefinition[1].contents)
	if len(errs) != 0 {
		return errs
	}

	// Compile the function
	assembly, errs := state.compileBlockToAssembly(functionDefinition[2:], regState, siblingFunctions, assemblyForControlFlowKeywords{})
	if len(errs) != 0 {
		return errs
	}
	assert(notEq(assembly, ""))
	if assembly[len(assembly)-1] != '\\' {
		// If the compiled assembly does not return at the end, then add a return
		assembly += "\n\\"
	}
	state.compiledFunctions[functionName] = compiledFunction{assembly: assembly}

	// Return
	return []codeParsingError{}
}

func compileAssembly(AST []ASTitem) (string, []codeParsingError) {
	assert(notEq(len(AST), 0))

	// Get all of the globally declared functions in the AST
	globalFunctions := make(map[string][]ASTitem)
	for _, ASTitem := range AST {
		if ASTitem.itemType == Function {
			assert(eq(ASTitem.contents[1].itemType, Function))
			functionName := ASTitem.contents[1].name
			assert(notEq(functionName, ""))
			if globalFunctions[functionName] != nil {
				errMsg := errors.New("Two declarations of a function called `" + functionName +
					"`. Functions can only be declared once.")
				return "", []codeParsingError{
					{msg: errMsg, location: globalFunctions[ASTitem.name][0].location},
					{msg: errMsg, location: ASTitem.location},
				}
			}
			globalFunctions[functionName] = ASTitem.contents
		} else {
			// TODO: Handle AST items other then functions
		}
	}

	// Check that the main function exists
	if globalFunctions["main"] == nil {
		return "", []codeParsingError{{
			location: textLocation{
				line:   1,
				column: 1,
			},
			msg: errors.New("Could not find main function defintion"),
		}}
	}

	// Compile the main function into assembly that has `\` to return from
	// functions, and `/FUNCTION_NAME/` to call other functions.
	state := compilerState{compiledFunctions: make(map[string]compiledFunction)}
	errs := state.compileFunctionDefinition(globalFunctions["main"], globalFunctions)
	if len(errs) != 0 {
		return "", errs
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
	return out + "\n", []codeParsingError{}
}

func (state *compilerState) transformFunctionDefinitionIntoValidAssembly(functionName string, returnAssembly string) {
	functionDefinition, ok := state.compiledFunctions[functionName]
	assert(eq(ok, true))
	if functionDefinition.jumpLabel != "" {
		return
	}

	// Update the function jump label so that when we call
	// `getAssemblyForFunctionCall` if it calls this function, then the early
	// return above can return before this function calls
	// `getAssemblyForFunctionCall` again possibly starting an infinite loop.
	if functionName == "main" {
		functionDefinition.jumpLabel = "_start"
	} else {
		functionDefinition.jumpLabel = state.createNewJumpLabel()
	}
	state.compiledFunctions[functionName] = functionDefinition

	// Change `functionDefinition.assembly` so that it is valid assembly
	functionDefinition.assembly = "\n" + functionDefinition.jumpLabel + ":" + functionDefinition.assembly
	charecterIsSingleQuoteString := false
	for index := 0; index < len(functionDefinition.assembly); index++ {
		if functionDefinition.assembly[index] == '\'' {
			charecterIsSingleQuoteString = !charecterIsSingleQuoteString
		} else if !charecterIsSingleQuoteString {
			if functionDefinition.assembly[index] == '/' {
				startIndex := index
				index++
				assert(lessThan(index, len(functionDefinition.assembly)))
				for functionDefinition.assembly[index] != '/' {
					index++
					assert(lessThan(index, len(functionDefinition.assembly)))
				}
				functionDefinition.assembly =
					functionDefinition.assembly[:startIndex] +
						state.getAssemblyForFunctionCall(functionDefinition.assembly[startIndex+1:index]) +
						functionDefinition.assembly[index+1:]
			} else if functionDefinition.assembly[index] == '\\' {
				functionDefinition.assembly = functionDefinition.assembly[:index] + returnAssembly + functionDefinition.assembly[index+1:]
			}
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

// Gets the register from a variable's AST item, and if the variable's AST item is of type
// DropVariable, then this function also handles dropping the variables
func getRegisterFromVariableASTitem(regState *registerState, variableASTitem ASTitem) (int, codeParsingError) {
	assert(or(eq(variableASTitem.itemType, Name), eq(variableASTitem.itemType, DropVariable)))
	assert(notEq(variableASTitem.name, ""))

	// Find the variable's register
	register := 0
	for regState.registers[register].variableName != variableASTitem.name {
		register++
		if register >= len(regState.registers) {
			return -1, codeParsingError{
				location: variableASTitem.location,
				msg:      errors.New("Could not find a variable called " + variableASTitem.name),
			}
		}
	}

	// Early return if we don't have to handle dropping the variable
	if variableASTitem.itemType == Name {
		return register, codeParsingError{}
	}

	// Check that dropping the variable is valid
	if regState.registers[register].stopVariableFromBeingDropped {
		return -1, codeParsingError{
			location: variableASTitem.location,
			msg:      errors.New("You cannot drop the `" + variableASTitem.name + "` variable in this scope since the variable is declared outside this scope"),
		}
	}
	if regState.registers[register].registerWasDefinedAsMutatableAt.line == 0 {
		return -1, codeParsingError{
			location: variableASTitem.location,
			msg: errors.New("Without this register (r" + fmt.Sprint(register) + ") being mutatable, " +
				"you cannot reserve this register for another variable after you have dropped the old " +
				"variable ( `" + variableASTitem.name + "`) at this line of code. You also won't be able to mutate " +
				"this register after you have dropped it. So there is no point in dropping this variable."),
		}
	}

	// Drop the variable
	regState.registers[register].variableName = ""
	regState.registers[register].variableNameWasDefinedAt = textLocation{}

	// Return
	return register, codeParsingError{}
}

// Parses any value that can go on the right side of an equals into assembly
func (state *compilerState) convertValueToAssembly(regState *registerState, value ASTitem) (string, codeParsingError) {
	switch value.itemType {
	// TODO: Add support for floats
	// We do not need to handle `&variableName` since variables are registers, and it is not possible to have a pointer to a register
	case IntNumber:
		return "$" + value.name, codeParsingError{}
	case Name, DropVariable:
		registerNumber, err := getRegisterFromVariableASTitem(regState, value)
		if err.msg != nil {
			return "", err
		}
		return commonAssemblyRegisterToX86Register(registerNumber), codeParsingError{}
	case Dereference:
		assert(eq(len(value.contents), 1))
		switch value.contents[0].itemType {
		case Name, Dereference, DropVariable:
			assembly, err := state.convertValueToAssembly(regState, value.contents[0])
			return "(" + assembly + ")", err
		default:
			panic("Unexpected internal state: `value.contents[0].itemType` equals " + value.contents[0].itemType.String())
		}
	case StringValue:
		dataSectionLabelForString := state.createNewDataSectionLabel()
		state.dataSection += "\n" + dataSectionLabelForString + ": .ascii \"" + value.name + "\""
		return "$" + dataSectionLabelForString, codeParsingError{}
	case CharValue:
		return "$'" + value.name + "'", codeParsingError{}
	default:
		return "", codeParsingError{
			location: value.location,
			msg:      errors.New("Expected a value, got an AST item of type " + value.itemType.String()),
		}
	}
}

func isValidLastOperandForMoveAndCmpInstructions(itemType typeOfParsedKeywordOrASTitem) bool {
	// In AT&T assembly syntax, the second operator for the cmp, and the mov instructions must either
	// be a register or a memory operand
	return itemType == Name || itemType == DropVariable || itemType == Dereference
}

// `jumpToOnTrue` and `jumpToOnFalse` can be blank strings, which means that the assembly should just continue executing if the conditions evauluates to that
func (state *compilerState) conditionToAssembly(regState *registerState, condition ASTitem, jumpToOnTrue string, jumpToOnFalse string) (string, codeParsingError) {
	assert(or(notEq(jumpToOnTrue, ""), notEq(jumpToOnFalse, "")))
	switch condition.itemType {
	case BoolValue:
		if condition.name == "true" {
			if jumpToOnTrue == "" {
				return "", codeParsingError{}
			} else {
				return "\njmp " + jumpToOnTrue, codeParsingError{}
			}
		} else if condition.name == "false" {
			if jumpToOnFalse == "" {
				return "", codeParsingError{}
			} else {
				return "\njmp " + jumpToOnFalse, codeParsingError{}
			}
		} else {
			panic("Unexpected internal state: Expected condition.name to equal `true`, or `false`, got `" + condition.name + "`")
		}
	case And, Or:
		out := ""
		afterConditionJumpLabel := state.createNewJumpLabel()
		jumpToOnClauseTrue := ""
		jumpToOnClauseFalse := ""
		if condition.itemType == And {
			// In an and condition, if any clause is false, then the whole clause is false
			if jumpToOnFalse != "" {
				jumpToOnClauseFalse = jumpToOnFalse
			} else {
				jumpToOnClauseFalse = afterConditionJumpLabel
			}
		} else if condition.itemType == Or {
			// In an or condition, if any clause is true, then the whole clause is true
			if jumpToOnTrue != "" {
				jumpToOnClauseTrue = jumpToOnTrue
			} else {
				jumpToOnClauseTrue = afterConditionJumpLabel
			}
		}
		for i, clause := range condition.contents {
			if i == len(condition.contents)-1 {
				jumpToOnClauseFalse = jumpToOnFalse
				jumpToOnClauseTrue = jumpToOnTrue
			}
			assembly, err := state.conditionToAssembly(regState, clause, jumpToOnClauseTrue, jumpToOnClauseFalse)
			if err.msg != nil {
				return "", err
			}
			out += assembly
		}
		return out + "\n" + afterConditionJumpLabel + ":", codeParsingError{}
	case ComparisonSyntax:
		out := ""
		assert(eq(len(condition.contents), 2))
		if !isValidLastOperandForMoveAndCmpInstructions(condition.contents[1].itemType) {
			// In AT&T assembly syntax, the second operator for the cmp instruction must
			// either be a register or a memory operand, so we need need to flip the
			// operators, and the greater then sign.
			if !isValidLastOperandForMoveAndCmpInstructions(condition.contents[0].itemType) {
				return "", codeParsingError{
					msg:      errors.New("Comparisons must have at least 1 variable name or pointer to memory in them"),
					location: condition.location,
				}
			}
			condition.contents = []ASTitem{condition.contents[1], condition.contents[0]}
			if condition.name[0] == '<' {
				condition.name = ">" + condition.name[1:]
			} else if condition.name[0] == '>' {
				condition.name = "<" + condition.name[1:]
			}
		}
		firstArg, err := state.convertValueToAssembly(regState, condition.contents[0])
		if err.msg != nil {
			return "", err
		}
		secondArg, err := state.convertValueToAssembly(regState, condition.contents[1])
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
		default:
			panic("Unexpected internal state: Expected AST item of type ValueComparison to have name either >, >=, <, <=, ==, or !=, got " + condition.name)
		}

		if jumpToOnTrue != "" {
			out += "\n" + jumpOnTrueCmp + " " + jumpToOnTrue
			if jumpToOnFalse != "" {
				out += "\njmp " + jumpToOnFalse
			}
		} else if jumpToOnFalse != "" {
			out += "\n" + jumpOnFalseCmp + " " + jumpToOnFalse
		}
		return out, codeParsingError{}
	default:
		panic("Unexpected internal state: Expected condition.itemType to equal Bool, BooleanLogic, or ValueComparison, got " + condition.itemType.String())
	}
}
