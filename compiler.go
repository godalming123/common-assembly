package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Compiler.go
// ===========
// Responsible for compiling an abstract syntax tree into assembly

type individualRegisterState struct {
	// If the variableName == "", then this register is not assigned to a variable
	variableName string

	// Stores if a variable can be dropped in the current scope, for example if you define a variable
	// outside a while loop, then it cannot be dropped inside the while loop.
	stopVariableFromBeingDropped bool

	// A variable name can be defined at a different location to where the register was defined as
	// mutable, for example:
	// fn b0 = myFunc() {
	//  # ^ register was defined as mutable here
	//     b0 name = "John"
	//      # ^ variable name was defined here
	// }

	// This location is saved so that the compiler can warn the user if a variable is not used
	variableNameWasDefinedAt textLocation

	// This location is saved so that the compiler can warn the user if a register that was defined as
	// mutable is not mutated. If registerWasDefinedAsMutableAt == textLocation{}, then this
	// register was not defined as mutable.
	registerWasDefinedAsMutableAt textLocation
}

type registerState struct {
	registers [16]individualRegisterState

	// The registers that the surrounding function uses to return values to the caller. This is a
	// subset of the registers that the surroinding function can mutate.
	functionReturnValueRegisters []Register
}

func commonAssemblyRegisterToX86Register(registerIndex Register) string {
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
	block []statement,
	regState registerState,
	siblingFunctions map[string]functionDefinition,
	controlFlowKeywordsAssembly assemblyForControlFlowKeywords,
) (string, []codeParsingError) {
	assembly := ""
	for index, genericStatement := range block {
		switch statement := genericStatement.(type) {

		case comment:

		case returnStatement:
			assert(eq(index, len(block)-1))
			assemblyForArgs, returnRegisters, errs := state.compileFunctionCallArguments(statement.returnedValues, &regState, false)
			if len(errs) != 0 {
				return "", errs
			}
			err := checkRegisterListsAreTheSame(regState.functionReturnValueRegisters, returnRegisters)
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			return assembly + assemblyForArgs + "\n\\", []codeParsingError{}

		case mutationStatement:
			assemblyForStatement := ""
			errs := []codeParsingError{}
			switch operation := statement.operation.(type) {
			case setToFunctionCallValue:
				assemblyForStatement, errs = state.compileFunctionCall(statement.destination, operation, &regState, siblingFunctions)
			case incrementBy1:
				assemblyForStatement, errs = state.compileVariableMutation("inc", nil, statement.destination, statement.textLocation, &regState)
			case decrementBy1:
				assemblyForStatement, errs = state.compileVariableMutation("dec", nil, statement.destination, statement.textLocation, &regState)
			case setToRawValue:
				assemblyForStatement, errs = state.compileVariableMutation("mov", operation.val, statement.destination, statement.textLocation, &regState)
			case incrementByRawValue:
				assemblyForStatement, errs = state.compileVariableMutation("add", operation.val, statement.destination, statement.textLocation, &regState)
			case decrementByRawValue:
				assemblyForStatement, errs = state.compileVariableMutation("sub", operation.val, statement.destination, statement.textLocation, &regState)
			case multiplyByRawValue:
				assemblyForStatement, errs = state.compileVariableMutation("mul", operation.val, statement.destination, statement.textLocation, &regState)
			case divideByRawValue:
				assemblyForStatement, errs = state.compileVariableMutation("div", operation.val, statement.destination, statement.textLocation, &regState)
			default:
				panic("Unexpected internal state:\n" +
					"- Expected `statement.operation.(type)` to be equal to either:\n" +
					"  - `setToFunctionCallValue`\n" +
					"  - `incrementBy1`\n" +
					"  - `decrementBy1`\n" +
					"  - `setToRawValue`\n" +
					"  - `incrementByRawValue`\n" +
					"  - `decrementByRawValue`\n" +
					"  - `multiplyByRawValue`\n" +
					"  - `divideByRawValue`\n" +
					"- But it equals `" + fmt.Sprint(reflect.TypeOf(statement.operation)) + "`\n" +
					"- Context: `statement.line` is " + fmt.Sprint(statement.line) + "\n" +
					"- Context: `statement.column` is " + fmt.Sprint(statement.column),
				)
			}
			if len(errs) != 0 {
				return "", errs
			}
			assembly += assemblyForStatement

		case whileLoop:
			// Save jump labels
			loopBodyJumpLabel := state.createNewJumpLabel()
			loopConditionJumpLabel := state.createNewJumpLabel()
			loopEndJumpLabel := state.createNewJumpLabel()

			// Add loop head
			assembly += "\njmp " + loopConditionJumpLabel

			// Add loop body
			assembly += "\n" + loopBodyJumpLabel + ":"
			loopBodyAssembly, errs := state.compileBlockToAssembly(
				statement.loopBody,
				parseRegisterStatesToInnerScope(regState),
				siblingFunctions,
				assemblyForControlFlowKeywords{
					breakAssembly:    "\njmp " + loopEndJumpLabel,
					continueAssembly: "\njmp " + loopConditionJumpLabel,
				},
			)
			if len(errs) != 0 {
				return "", errs
			}
			assembly += loopBodyAssembly

			// Add loop condition
			assembly += "\n" + loopConditionJumpLabel + ":"
			conditionAssembly, err := state.conditionToAssembly(&regState,
				statement.condition, loopBodyJumpLabel, "")
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			assembly += conditionAssembly

			// Add loop end
			assembly += "\n" + loopEndJumpLabel + ":"

		case ifElseStatement:
			elseBlockJumpLabel := state.createNewJumpLabel()
			ifCheck, err := state.conditionToAssembly(&regState,
				statement.condition, "", elseBlockJumpLabel)
			if err.msg != nil {
				return "", []codeParsingError{err}
			}
			innerScopeRegStates := parseRegisterStatesToInnerScope(regState)
			ifBody, errs := state.compileBlockToAssembly(statement.ifBlock,
				innerScopeRegStates, siblingFunctions, controlFlowKeywordsAssembly)
			if len(errs) != 0 {
				return "", errs
			}
			if len(statement.elseBlock) > 0 {
				endJumpLabel := state.createNewJumpLabel()
				elseBody, errs := state.compileBlockToAssembly(statement.elseBlock,
					innerScopeRegStates, siblingFunctions, controlFlowKeywordsAssembly)
				if len(errs) != 0 {
					return "", errs
				}
				assembly += ifCheck + ifBody + "\njmp " + endJumpLabel + "\n" +
					elseBlockJumpLabel + ":" + elseBody + "\n" + endJumpLabel + ":"
			} else {
				assembly += ifCheck + ifBody + "\n" + elseBlockJumpLabel + ":"
			}

		case breakStatement:
			if controlFlowKeywordsAssembly.breakAssembly == "" {
				return "", []codeParsingError{{
					msg:          errors.New("Break statement is not valid in this scope"),
					textLocation: textLocation(statement),
				}}
			}
			assembly += controlFlowKeywordsAssembly.breakAssembly
		case continueStatement:
			if controlFlowKeywordsAssembly.continueAssembly == "" {
				return "", []codeParsingError{{
					msg:          errors.New("Continue statement is not valid in this scope"),
					textLocation: textLocation(statement),
				}}
			}
			assembly += controlFlowKeywordsAssembly.continueAssembly

		case dropVariableStatement:
			_, err := getRegisterFromVariableName(&regState, statement.variable, true, statement.textLocation)
			if err.msg != nil {
				return "", []codeParsingError{err}
			}

		default:
			panic("Unexpected internal state")
		}
	}
	return assembly, []codeParsingError{}
}

type registerAndLocation struct {
	register Register
	location textLocation
}

// Compiles the arguments in a function call into assembly
func (state *compilerState) compileFunctionCallArguments(
	functionArguments []registerAndRawValueAndLocation,
	regState *registerState,

	// Whether to check for if a variable is mutated without naming the variable by naming the register
	// associated with the variable. This is turned off when this function is used to compile return
	// statements, since in that case it does not matter if a variable declared in the function is
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
		argRegister := arg.register
		if argRegister == UnknownRegister {
			variableParsed, isVariable := arg.value.(variableValue)
			if !isVariable {
				return "", []registerAndLocation{}, []codeParsingError{{
					msg: errors.New("If you don't specify which register to use, you must " +
						"pass a variable. This argument does not specify which register to use " +
						"and passes a value of type " + fmt.Sprint(reflect.TypeOf(arg.value))),
					textLocation: arg.textLocation,
				}}
			}
			var err codeParsingError
			argRegister, err = getRegisterFromVariableName(regState,
				variableParsed.name, variableParsed.variableIsDropped,
				variableParsed.textLocation)
			if err.msg != nil {
				return "", []registerAndLocation{}, []codeParsingError{err}
			}
		} else {
			if regState.registers[argRegister].registerWasDefinedAsMutableAt.line == 0 {
				return "", []registerAndLocation{}, []codeParsingError{{
					textLocation: arg.textLocation,
					msg:          errors.New("It is not possible to mutate the register r" + fmt.Sprint(argRegister) + "."),
				}}
			}

			if checkImplicitVariableMutation && regState.registers[argRegister].variableName != "" {
				return "", []registerAndLocation{}, []codeParsingError{{
					textLocation: arg.textLocation,
					msg:          errors.New("It is only possible to mutate the register r" + fmt.Sprint(argRegister) + " through the variable " + regState.registers[argRegister].variableName),
				}}
			}

			argValue, err := state.convertValueToAssembly(regState, arg.value)
			if err.msg != nil {
				return "", []registerAndLocation{}, []codeParsingError{err}
			}

			assembly += "\nmov " + argValue + ", " + commonAssemblyRegisterToX86Register(argRegister)
		}

		for _, register := range registers {
			if register.register == argRegister {
				errMsg := errors.New("Register r" + fmt.Sprint(register) + " used atleast twice in function arguments. Each register can only be used once.")
				return "", []registerAndLocation{}, []codeParsingError{
					{msg: errMsg, textLocation: register.location},
					{msg: errMsg, textLocation: arg.textLocation},
				}
			}
		}
		add(&registers, registerAndLocation{
			register: argRegister,
			location: arg.textLocation,
		})
	}
	return assembly, registers, []codeParsingError{}
}

// Checks that a variable mutation destination for the following errors:
// - A register that is reserved for a variable is implicityly mutated without naming the variable
// - An undefined variable has been mutated
// - A defined variable has been re-defined
// - A register that the surrounding function does not mark as mutable has been mutated
func validateVariableMutationDestination(mutatedValue variableMutationDestination, regState *registerState) (Register, []codeParsingError) {
	// Initialise variables
	registerTheVariableWasAlreadyDefinedToUse := UnknownRegister
	errs := []codeParsingError{}

	// Handle the mutated register if there is one
	if mutatedValue.register != UnknownRegister {
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
				textLocation: mutatedValue.textLocation,
			})
		}
	}

	// Handle the mutated variable if there is one
	if mutatedValue.name != "" {
		// Get the register that was already defined to use this variable if there is one
		for variableRegister, variable := range regState.registers {
			if variable.variableName == mutatedValue.name {
				registerTheVariableWasAlreadyDefinedToUse = Register(variableRegister)
				break
			}
		}

		// Check possible errors
		if registerTheVariableWasAlreadyDefinedToUse == -1 {
			if mutatedValue.register == -1 {
				// The user has tried to mutate a variable that has not been defined yet
				add(&errs, codeParsingError{
					msg: errors.New("You have tried to mutate a variable (`" + mutatedValue.name + "`) that has not " +
						"been defined yet. If you want to define this variable, then add the register that this" +
						" variable will use next to the variable name."),
					textLocation: mutatedValue.textLocation,
				})
			}
		} else if mutatedValue.register != -1 {
			// The user has tried to re-define a variable that is already defined
			if registerTheVariableWasAlreadyDefinedToUse != mutatedValue.register {
				add(&errs, codeParsingError{
					msg: errors.New("`" + mutatedValue.name + "` is already defined as using the register r" +
						fmt.Sprint(registerTheVariableWasAlreadyDefinedToUse) + ", however here you are trying to " +
						" redefine this variable to use a different register (r" + fmt.Sprint(mutatedValue.register) + "). If " +
						"you want to stop using the old variable, then add `drop" + mutatedValue.name + "` before this " +
						"line of code. If you want to use both variables, then you will have to change the name of " +
						"one of the variables."),
					textLocation: mutatedValue.textLocation,
				})
			} else {
				add(&errs, codeParsingError{
					msg: errors.New("Variable ( " + mutatedValue.name + ") and register (r" + fmt.Sprint(mutatedValue.register) +
						") named to mutate a variable that is already defined. After a variable has been " +
						"defined, it can be mutated by just naming the variable instead of naming the variable " +
						"and the register."),
					textLocation: mutatedValue.textLocation,
				})
			}
		}
	}

	// Get the register the user mutated
	register := mutatedValue.register
	if register == UnknownRegister {
		assert(notEq(registerTheVariableWasAlreadyDefinedToUse, -1))
		register = registerTheVariableWasAlreadyDefinedToUse
	} else {
		assert(eq(registerTheVariableWasAlreadyDefinedToUse, -1))
	}

	// Handle if the user tried to mutate a register that the surrounding function does not explicitly
	// mutate
	if regState.registers[register].registerWasDefinedAsMutableAt.line == 0 {
		add(&errs, codeParsingError{
			msg: errors.New("You cannot mutate the r" + fmt.Sprint(register) + " register unless you add " +
				"it to the list of registers that the function mutates."),
			textLocation: mutatedValue.textLocation,
		})
	}

	// Return if there are errors
	if len(errs) != 0 {
		return UnknownRegister, errs
	}

	// Handle updating register states
	if mutatedValue.register != -1 && mutatedValue.name != "" {
		assert(eq(regState.registers[register].variableName, ""))
		assert(eq(regState.registers[register].variableNameWasDefinedAt, textLocation{}))
		regState.registers[register].variableName = mutatedValue.name
		regState.registers[register].variableNameWasDefinedAt = mutatedValue.textLocation
	}

	// Return
	return register, []codeParsingError{}
}

// Compiles a variableMutation ASTitem of type Assignment, PlusEquals, MinusEquals, MultiplyEquals or DivideEquals into assembly
func (state *compilerState) compileVariableMutation(
	instruction string,
	source rawValue,
	destination []variableMutationDestination,
	location textLocation,
	regState *registerState,
) (string, []codeParsingError) {
	// Check that there is only one thing be mutated
	if len(destination) != 1 {
		return "", []codeParsingError{{
			textLocation: location,
			msg: errors.New(
				"Expect 1 value on left side of equals unless a function is being called. Got " +
					fmt.Sprint(len(destination)) +
					" values on left side of equals being set to one value of type " +
					fmt.Sprint(reflect.TypeOf(source)) + ". Please split this into multiply assignments.",
			),
		}}
	}

	// Get the common assembly register that is being mutated, and update the register states
	register, errs := validateVariableMutationDestination(destination[0], regState)
	if len(errs) != 0 {
		return "", errs
	}

	// Check that the register is reserved for a variable
	if destination[0].name == "" {
		return "", []codeParsingError{{
			textLocation: destination[0].textLocation,
			msg: errors.New("Without giving a register a variable name, the value that" +
				" you assign to the register here cannot be used later, so there is no" +
				" point in assigning a value to a register without reserving the register" +
				" for use with a specific variable."),
		}}
	}

	// Convert the common assembly register number into an x86 register
	mutatedRegisterAssembly := strings.Repeat(
		"(", int(destination[0].pointerDereferenceLayers)) +
		commonAssemblyRegisterToX86Register(register) +
		strings.Repeat(")", int(destination[0].pointerDereferenceLayers))

	// Get the assembly for the source if a source is specified
	if source == nil {
		return "\n" + instruction + " " + mutatedRegisterAssembly, []codeParsingError{}
	} else {
		valueBeingAssignedToVariable, err := state.convertValueToAssembly(regState, source)
		if err.msg != nil {
			return "", []codeParsingError{err}
		}
		return "\n" + instruction + " " + valueBeingAssignedToVariable + ", " + mutatedRegisterAssembly, []codeParsingError{}
	}
}

// Compiles a functionCall ASTitem of type Assignment, PlusEquals, MinusEquals, MultiplyEquals or DivideEquals into assembly
func (state *compilerState) compileFunctionCall(
	destination []variableMutationDestination,
	operation setToFunctionCallValue,
	regState *registerState,
	siblingFunctions map[string]functionDefinition,
) (string, []codeParsingError) {
	// TODO: Add support for functions having any as a register
	assert(notEq(operation.functionName, ""))

	// Check that the function is defined, and get the code to call the function
	functionCallCode := ""
	_, isUserDefinedFunction := siblingFunctions[operation.functionName]
	if isUserDefinedFunction {
		// Compile the function if it has not been compiled already
		errs := state.compileFunctionDefinition(siblingFunctions[operation.functionName], siblingFunctions)
		if len(errs) != 0 {
			return "", errs
		}

		// Increase the references to the function
		entry, ok := state.compiledFunctions[operation.functionName]
		assert(eq(ok, true))
		entry.references++
		state.compiledFunctions[operation.functionName] = entry

		// Set functionCallCode
		functionCallCode = "/" + operation.functionName + "/"
	} else {
		switch operation.functionName {
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
				textLocation: operation.textLocation,
				msg:          errors.New("Call to undefined function `" + operation.functionName + "`"),
			}}
		}
	}

	// Compile the function arguments
	assemblyForArgs, functionCallArgRegisters, errs := state.compileFunctionCallArguments(operation.functionArgs, regState, true)
	if len(errs) != 0 {
		return "", errs
	}

	// Get the expected registers of the function arguments
	functionExpectedArgRegisters := []Register{}
	if isUserDefinedFunction {
		functionExpectedArgRegisters = mapList(
			siblingFunctions[operation.functionName].arguments,
			func(r registerAndNameAndLocation) Register {
				assert(notEq(r.register, UnknownRegister))
				return r.register
			},
		)
	} else {
		switch operation.functionName {
		case "sysRead", "sysWrite", "sysOpen":
			functionExpectedArgRegisters = []Register{5, 4, 3}
		case "sysClose", "sysBrk", "sysExit":
			functionExpectedArgRegisters = []Register{5}
		default:
			panic("Unexpected internal state: isUserDefinedFunction is false, and functionName is `" + operation.functionName + "`.")
		}
	}

	// Check that the function arguments use the expected registers
	err := checkRegisterListsAreTheSame(functionExpectedArgRegisters, functionCallArgRegisters)
	if err.msg != nil {
		return "", []codeParsingError{err}
	}

	// Get mutated registers
	errs = []codeParsingError{}
	for i, mutatedRegister := range destination {
		addedErrs := []codeParsingError{}
		destination[i].register, addedErrs = validateVariableMutationDestination(mutatedRegister, regState)
		add(&errs, addedErrs...)
		if mutatedRegister.pointerDereferenceLayers > 0 {
			add(&errs, codeParsingError{
				msg:          errors.New("Mutated value in function call cannot be dereferenced with ^"),
				textLocation: mutatedRegister.textLocation,
			})
		}
	}
	if len(errs) > 0 {
		return "", errs
	}

	// Get the expected mutated registers
	functionExpectedMutatedRegisters := []registerAndNameAndLocation{}
	if isUserDefinedFunction {
		functionExpectedMutatedRegisters = siblingFunctions[operation.functionName].mutatedRegisters
	} else {
		switch operation.functionName {
		case "sysRead", "sysWrite", "sysClose", "sysBrk", "sysExit":
			functionExpectedMutatedRegisters = []registerAndNameAndLocation{{
				register: 0,
				name:     "exitCode",
			}}
		case "sysOpen":
			functionExpectedMutatedRegisters = []registerAndNameAndLocation{{
				register: 0,
				name:     "fileDescriptor",
			}}
		default:
			panic("Unexpected internal state: isUserDefinedFunction is false, and functionName is `" + operation.functionName + "`.")
		}
	}

	// Check that the function mutated regisers use the expected registers
	err = checkRegisterListsAreTheSame(
		mapList(functionExpectedMutatedRegisters, func(in registerAndNameAndLocation) Register {
			return in.register
		}),
		mapList(destination, func(in variableMutationDestination) registerAndLocation {
			return registerAndLocation{
				register: in.register,
				location: in.textLocation,
			}
		}),
	)
	if err.msg != nil {
		return "", []codeParsingError{err}
	}
	for i, expectedMutatedRegister := range functionExpectedMutatedRegisters {
		if expectedMutatedRegister.name == "" && destination[i].name != "" {
			return "", []codeParsingError{{
				textLocation: destination[i].textLocation,
				msg: errors.New("Function call stores the final value that the r" +
					fmt.Sprint(destination[i].register) + " register was mutated to in a new " +
					"variable called `" + destination[i].name + "`, but the function " +
					"definition does not guarantee that it will muatate that register."),
			}}
		}
	}

	// Return
	return assemblyForArgs + "\n" + functionCallCode, []codeParsingError{}
}

func parseFunctionDefinitionRegisters(
	mutatedRegisters []registerAndNameAndLocation,
	functionArgs []registerAndNameAndLocation,
) (registerState, []codeParsingError) {
	out := registerState{}

	// Parse the function mutated registers
	for _, register := range mutatedRegisters {
		// TODO: Check that the mutated registers do not have the same name
		assert(notEq(register.register, -1))
		if register.name != "" {
			add(&out.functionReturnValueRegisters, register.register)
		}
		if out.registers[register.register].registerWasDefinedAsMutableAt.line != 0 {
			errMsg := errors.New("Register " + register.name + " used twice in mutated registers")
			return registerState{}, []codeParsingError{
				{msg: errMsg, textLocation: out.registers[register.register].registerWasDefinedAsMutableAt},
				{msg: errMsg, textLocation: register.textLocation},
			}
		}
		out.registers[register.register].registerWasDefinedAsMutableAt = register.textLocation
	}

	// Parse the function args
	for _, arg := range functionArgs {
		assert(notEq(arg.register, -1))
		assert(notEq(arg.name, ""))

		// Check that the same register has not been used already
		if out.registers[arg.register].variableName != "" {
			errMsg := errors.New("Register " + arg.name + " used twice in function arguments. Each register can only be used once.")
			return registerState{}, []codeParsingError{
				{msg: errMsg, textLocation: out.registers[arg.register].variableNameWasDefinedAt},
				{msg: errMsg, textLocation: arg.textLocation},
			}
		}

		// Check that the same name has not been used already
		for _, regState := range out.registers {
			if regState.variableName == arg.name {
				errMsg := errors.New("Variable name " + arg.name + " used twice in function arguments. Each variable name can only be used once.")
				return registerState{}, []codeParsingError{
					{msg: errMsg, textLocation: out.registers[arg.register].variableNameWasDefinedAt},
					{msg: errMsg, textLocation: arg.textLocation},
				}
			}
		}

		// Update registerStates
		out.registers[arg.register].variableName = arg.name
		out.registers[arg.register].variableNameWasDefinedAt = arg.textLocation
	}

	// Return
	return out, []codeParsingError{}
}

func (state *compilerState) compileFunctionDefinition(
	function functionDefinition,
	siblingFunctions map[string]functionDefinition,
) []codeParsingError {
	assert(notEq(function.name, ""))

	// Add the `functionName` key to the compiledFunctions hashmap so that when
	// `compileBlockToAssembly()` calls `compileFunctionCall`, that does not call
	// this function if the function being called is the current function being
	// compiled to stop an infinite loop.
	state.compiledFunctions[function.name] = compiledFunction{}

	// Parse registers that the function mutates
	regState, errs := parseFunctionDefinitionRegisters(function.mutatedRegisters, function.arguments)
	if len(errs) != 0 {
		return errs
	}

	// Compile the function
	assembly, errs := state.compileBlockToAssembly(function.body, regState, siblingFunctions, assemblyForControlFlowKeywords{})
	if len(errs) != 0 {
		return errs
	}
	assert(notEq(assembly, ""))
	if assembly[len(assembly)-1] != '\\' {
		// If the compiled assembly does not return at the end, then add a return
		assembly += "\n\\"
	}
	state.compiledFunctions[function.name] = compiledFunction{assembly: assembly}

	// Return
	return []codeParsingError{}
}

func compileAssembly(AST []topLevelASTitem) (string, []codeParsingError) {
	// Get all of the globally declared functions in the AST
	globalFunctions := make(map[string]functionDefinition)
	for _, ASTitem := range AST {
		function, ok := ASTitem.(functionDefinition)
		if !ok {
			continue
		}
		assert(notEq(function.name, ""))
		if _, exists := globalFunctions[function.name]; exists {
			errMsg := errors.New("Two declarations of a function called `" + function.name +
				"`. Functions can only be declared once.")
			return "", []codeParsingError{
				{msg: errMsg, textLocation: globalFunctions[function.name].textLocation},
				{msg: errMsg, textLocation: function.textLocation},
			}
		}
		globalFunctions[function.name] = function
	}

	// Check that the main function exists
	if _, exists := globalFunctions["main"]; !exists {
		return "", []codeParsingError{{
			textLocation: textLocation{
				line:   1,
				column: 1,
			},
			msg: errors.New("Could not find main function definition"),
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

	// Concatenate the output
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
	characterIsSingleQuoteString := false
	for index := 0; index < len(functionDefinition.assembly); index++ {
		if functionDefinition.assembly[index] == '\'' {
			characterIsSingleQuoteString = !characterIsSingleQuoteString
		} else if !characterIsSingleQuoteString {
			switch functionDefinition.assembly[index] {
			case '/':
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
			case '\\':
				functionDefinition.assembly = functionDefinition.assembly[:index] +
					returnAssembly + functionDefinition.assembly[index+1:]
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

// Gets the register from a variable's name, and if `variableIsDropped == true`,
// then this function also handles dropping the variables.
func getRegisterFromVariableName(
	regState *registerState,
	variableName string,
	variableIsDropped bool,
	variableLocation textLocation,
) (Register, codeParsingError) {
	assert(notEq(variableName, ""))

	// Find the variable's register
	register := Register(0)
	for regState.registers[register].variableName != variableName {
		register++
		if register >= Register(len(regState.registers)) {
			return UnknownRegister, codeParsingError{
				textLocation: variableLocation,
				msg:          errors.New("Could not find a variable called `" + variableName + "`"),
			}
		}
	}

	// Early return if we don't have to handle dropping the variable
	if !variableIsDropped {
		return register, codeParsingError{}
	}

	// Check that dropping the variable is valid
	if regState.registers[register].stopVariableFromBeingDropped {
		return UnknownRegister, codeParsingError{
			textLocation: variableLocation,
			msg:          errors.New("You cannot drop the `" + variableName + "` variable in this scope since the variable is declared outside this scope"),
		}
	}
	if regState.registers[register].registerWasDefinedAsMutableAt.line == 0 {
		return UnknownRegister, codeParsingError{
			textLocation: variableLocation,
			msg: errors.New("Without this register (r" + fmt.Sprint(register) + ") being mutable, " +
				"you cannot reserve this register for another variable after you have dropped the old " +
				"variable ( `" + variableName + "`) at this line of code. You also won't be able to mutate " +
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
func (state *compilerState) convertValueToAssembly(regState *registerState, untypedValue rawValue) (string, codeParsingError) {
	switch value := untypedValue.(type) {
	// TODO: Add support for floats
	// We do not need to handle `&variableName` since variables are registers, and it is not possible to have a pointer to a register
	case numberValue[uint64]:
		return "$" + fmt.Sprint(value.value), codeParsingError{}
	case numberValue[int64]:
		return "$" + fmt.Sprint(value.value), codeParsingError{}
	case numberValue[float64]:
		return "$" + fmt.Sprint(value.value), codeParsingError{}
	case variableValue:
		registerNumber, err := getRegisterFromVariableName(regState, value.name,
			value.variableIsDropped, value.textLocation)
		if err.msg != nil {
			return "", err
		}
		return strings.Repeat("(", int(value.pointerDereferenceLayers)) +
			commonAssemblyRegisterToX86Register(registerNumber) +
			strings.Repeat(")", int(value.pointerDereferenceLayers)), codeParsingError{}
	case stringValue:
		dataSectionLabelForString := state.createNewDataSectionLabel()
		state.dataSection += "\n" + dataSectionLabelForString + ": .ascii \"" + value.value + "\""
		return "$" + dataSectionLabelForString, codeParsingError{}
	case characterValue:
		return "$'" + value.value + "'", codeParsingError{}
	default:
		panic("Unexpected internal state")
	}
}

func isValidLastOperandForMoveAndCmpInstructions(value rawValue) bool {
	// In AT&T assembly syntax, the second operator for the cmp, and the mov instructions must either
	// be a register or a memory operand
	_, isVariableValue := value.(variableValue)
	return isVariableValue
}

// `jumpToOnTrue` and `jumpToOnFalse` can be blank strings, which means that the
// assembly should just continue executing if the conditions evaluates to that.
func (state *compilerState) conditionToAssembly(
	regState *registerState,
	untypedCondition condition,
	jumpToOnTrue string,
	jumpToOnFalse string,
) (string, codeParsingError) {
	assert(or(notEq(jumpToOnTrue, ""), notEq(jumpToOnFalse, "")))
	switch condition := untypedCondition.(type) {

	case booleanValue:
		if condition.value {
			if jumpToOnTrue == "" {
				return "", codeParsingError{}
			} else {
				return "\njmp " + jumpToOnTrue, codeParsingError{}
			}
		} else {
			if jumpToOnFalse == "" {
				return "", codeParsingError{}
			} else {
				return "\njmp " + jumpToOnFalse, codeParsingError{}
			}
		}

	case boolean:
		out := ""
		afterConditionJumpLabel := state.createNewJumpLabel()
		jumpToOnClauseTrue := ""
		jumpToOnClauseFalse := ""
		if condition.isAndInsteadOfOr {
			// In an and condition, if any clause is false, then the whole clause is false
			if jumpToOnFalse != "" {
				jumpToOnClauseFalse = jumpToOnFalse
			} else {
				jumpToOnClauseFalse = afterConditionJumpLabel
			}
		} else {
			// In an or condition, if any clause is true, then the whole clause is true
			if jumpToOnTrue != "" {
				jumpToOnClauseTrue = jumpToOnTrue
			} else {
				jumpToOnClauseTrue = afterConditionJumpLabel
			}
		}
		for i, clause := range condition.conditions {
			if i == len(condition.conditions)-1 {
				jumpToOnClauseFalse = jumpToOnFalse
				jumpToOnClauseTrue = jumpToOnTrue
			}
			assembly, err := state.conditionToAssembly(regState, clause,
				jumpToOnClauseTrue, jumpToOnClauseFalse)
			if err.msg != nil {
				return "", err
			}
			out += assembly
		}
		return out + "\n" + afterConditionJumpLabel + ":", codeParsingError{}

	case comparison:
		out := ""
		if !isValidLastOperandForMoveAndCmpInstructions(condition.rightValue) {
			// In AT&T assembly syntax, the second operator for the cmp instruction must
			// either be a register or a memory operand, so we need need to flip the
			// operators, and the greater then sign.
			if !isValidLastOperandForMoveAndCmpInstructions(condition.leftValue) {
				return "", codeParsingError{
					msg:          errors.New("Comparisons must have at least 1 variable name or pointer to memory in them"),
					textLocation: condition.textLocation,
				}
			}
			condition.leftValue, condition.rightValue =
				condition.rightValue, condition.leftValue
			switch condition.operator {
			case LessThan:
				condition.operator = GreaterThan
			case GreaterThan:
				condition.operator = LessThan
			case LessThanOrEqual:
				condition.operator = GreaterThanOrEqual
			case GreaterThanOrEqual:
				condition.operator = LessThanOrEqual
			}
		}
		firstArg, err := state.convertValueToAssembly(regState, condition.leftValue)
		if err.msg != nil {
			return "", err
		}
		secondArg, err := state.convertValueToAssembly(regState, condition.rightValue)
		if err.msg != nil {
			return "", err
		}
		out += "\ncmp " + firstArg + ", " + secondArg

		var jumpOnTrueCmp, jumpOnFalseCmp string
		switch condition.operator {
		case GreaterThan:
			jumpOnTrueCmp = "jl"
			jumpOnFalseCmp = "jge"
		case GreaterThanOrEqual:
			jumpOnTrueCmp = "jle"
			jumpOnFalseCmp = "jg"
		case LessThan:
			jumpOnTrueCmp = "jg"
			jumpOnFalseCmp = "jle"
		case LessThanOrEqual:
			jumpOnTrueCmp = "jge"
			jumpOnFalseCmp = "jl"
		case Equal:
			jumpOnTrueCmp = "je"
			jumpOnFalseCmp = "jne"
		case NotEqual:
			jumpOnTrueCmp = "jne"
			jumpOnFalseCmp = "je"
		default:
			panic("Unexpected internal state")
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
		panic("Unexpected internal state")

	}
}
