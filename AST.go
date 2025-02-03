// AST.go
// ======
// Responsable for defining the type definitions of the abstract syntax tree.

package main

// HELPER TYPES //
// ============ //

// A register in assembly. This is a value between `-1` and `15` inclusive. `-1` represents no
// register.
type Register int8

const UnkownRegister Register = -1

// A register, a name, and a location. This is used to represent function arguments,
// and function mutated registers.
type registerAndNameAndLocation struct {
	textLocation
	register Register
	name     string
}

// Compares 2 raw values (currently this does not include boolean values)
type comparisonOperation uint8

const (
	UnknownComparisonOperation comparisonOperation = iota
	GreaterThan
	LessThan
	GreaterThanOrEqual
	LessThanOrEqual
	Equal
	NotEqual
)

type numberOf64Bits interface {
	int64 | uint64 | float64
}

// GROUPS OF AST ITEMS //
// =================== //

// Any AST item that can be at the top level of a file
type topLevelASTitem interface {
	isTopLevelASTitem()
	location() textLocation
}

func (_ comment) isTopLevelASTitem()            {}
func (_ functionDefinition) isTopLevelASTitem() {}

// Any AST item that can be a statement like a function call, or a comment
type statement interface {
	isStatementASTitem()
	location() textLocation
}

func (_ comment) isStatementASTitem()               {}
func (_ ifElseStatement) isStatementASTitem()       {}
func (_ whileLoop) isStatementASTitem()             {}
func (_ mutationStatement) isStatementASTitem()     {}
func (_ returnStatement) isStatementASTitem()       {}
func (_ breakStatement) isStatementASTitem()        {}
func (_ continueStatement) isStatementASTitem()     {}
func (_ dropVariableStatement) isStatementASTitem() {}

// Any AST item that can be easily converted into the source operand for assembly's `mov`
// instruction.
type rawValue interface {
	isRawValue()
	location() textLocation
}

func (_ variableValue) isRawValue()    {}
func (_ numberValue[any]) isRawValue() {}
func (_ stringValue) isRawValue()      {}
func (_ charecterValue) isRawValue()   {}

// Any AST item that evaluates to either true or false
type condition interface {
	isCondition()
	location() textLocation
}

func (_ comparison) isCondition()   {}
func (_ boolean) isCondition()      {}
func (_ booleanValue) isCondition() {}

// Mutation operation
type mutationOperation interface {
	isMutationOperation()
	location() textLocation
}

func (_ incrementBy1) isMutationOperation()           {}
func (_ decrementBy1) isMutationOperation()           {}
func (_ setToFunctionCallValue) isMutationOperation() {}
func (_ setToRawValue) isMutationOperation()          {}
func (_ incrementByRawValue) isMutationOperation()    {}
func (_ decrementByRawValue) isMutationOperation()    {}
func (_ multiplyByRawValue) isMutationOperation()     {}
func (_ divideByRawValue) isMutationOperation()       {}

// INDIVIDUAL AST ITEMS //
// ==================== //

// TODO: There is a lot of overlap between the following types:
// - `variableValue` (location + name + variableIsDropped + pointerDereferenceLayers)
// - `variableMutationDestination` (location + register + name + pointerDereferenceLayers)
// - `registerAndNameAndLocation` (used in function definition arguments and mutated registers)
// - `registerAndRawValueAndLocation` (used in function arguments and the return values of return statements)

type comment struct {
	textLocation
	contents string
}

type booleanValue struct {
	textLocation
	value bool
}

type numberValue[numberType numberOf64Bits] struct {
	textLocation
	value numberType
}

// Each charecter must be representable in 64 bits, since we assume that you can directly set
// registers to charecters.
type charecterValue struct {
	textLocation
	value string
}

type stringValue struct {
	textLocation
	value string
}

// A variable that is used as a value
type variableValue struct {
	textLocation
	name              string
	variableIsDropped bool
	// The number of times to modify the value the register points to rather then the register itself
	pointerDereferenceLayers uint
}

type registerAndRawValueAndLocation struct {
	textLocation
	register Register
	value    rawValue
}

type functionDefinition struct {
	textLocation
	name             string
	arguments        []registerAndNameAndLocation
	mutatedRegisters []registerAndNameAndLocation
	body             []statement
}

type variableMutationDestination struct {
	textLocation
	register Register
	name     string
	// The number of times to modify the value the register points to rather then the register itself
	pointerDereferenceLayers uint
}

// A statement that mutates a variable/register
type mutationStatement struct {
	textLocation
	destination []variableMutationDestination
	operation   mutationOperation
}

type dropVariableStatement struct {
	textLocation
	variable string
}

type ifElseStatement struct {
	textLocation
	condition condition
	ifBlock   []statement
	elseBlock []statement
}

type whileLoop struct {
	textLocation
	condition condition
	loopBody  []statement
}

type returnStatement struct {
	textLocation
	returnedValues []registerAndRawValueAndLocation
}

type breakStatement textLocation
type continueStatement textLocation

func (statement breakStatement) location() textLocation    { return textLocation(statement) }
func (statement continueStatement) location() textLocation { return textLocation(statement) }

type comparison struct {
	textLocation
	operator   comparisonOperation
	leftValue  rawValue
	rightValue rawValue
}

type boolean struct {
	textLocation
	isAndInsteadOfOr bool
	conditions       []condition
}

type incrementBy1 struct{ textLocation }
type decrementBy1 struct{ textLocation }

type setToFunctionCallValue struct {
	textLocation
	functionName string
	functionArgs []registerAndRawValueAndLocation
}

type setToRawValue struct{ val rawValue }
type incrementByRawValue struct{ val rawValue }
type decrementByRawValue struct{ val rawValue }
type multiplyByRawValue struct{ val rawValue }
type divideByRawValue struct{ val rawValue }

func (operation setToRawValue) location() textLocation       { return operation.val.location() }
func (operation incrementByRawValue) location() textLocation { return operation.val.location() }
func (operation decrementByRawValue) location() textLocation { return operation.val.location() }
func (operation multiplyByRawValue) location() textLocation  { return operation.val.location() }
func (operation divideByRawValue) location() textLocation    { return operation.val.location() }
