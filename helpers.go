package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// This type is used to represent the type of keyword that a keyword is, and the type of ASTitem
// that an ASTitem is. The table below shows what kind of values would be in `keyword.contents` and
// `ASTitem.name` if for a keyword, or ASTitem of that `typeOfParsedKeywordOrASTitem`.
//
//go:generate go run golang.org/x/tools/cmd/stringer -type=typeOfParsedKeywordOrASTitem
type typeOfParsedKeywordOrASTitem uint8

const (
	Unknown typeOfParsedKeywordOrASTitem = iota
	//                // keyword.contents                       // ASTitem.name                 //
	// -------------- // -------------------------------------- // ---------------------------- //
	Name              // myFuncName1, myVarName2                // myVarName2                   //
	Register          // b0, b1, b2..., s0, s1, s2...           // SAME                         //
	StringValue       // "Foo", "Bar"                           // Foo, Bar                     //
	CharValue         // 'a', '\n'                              // a, \n                        //
	BoolValue         // true, false                            // SAME                         //
	IntNumber         // 4, 23                                  // SAME                         //
	FloatNumber       // 2.1, 5.8                               // SAME                         //
	IncreaseNesting   // (, {, [                                // INVALID                      //
	DecreaseNesting   // ), }, ]                                // INVALID                      //
	Function          // fn                                     // myFunctionName               //
	FunctionReturn    // return                                 // NONE                         //
	FunctionArgs      // INVALID                                // NONE                         //
	SetToAValue       // INVALID                                // NONE                         //
	DropVariable      // drop                                   // NONE                         //
	Assignment        // =                                      // NONE                         //
	Increment         // ++                                     // NONE                         //
	Decrement         // --                                     // NONE                         //
	PlusEquals        // +=                                     // NONE                         //
	MinusEquals       // -=                                     // NONE                         //
	MultiplyEquals    // *=                                     // NONE                         //
	DivideEquals      // /=                                     // NONE                         //
	WhileLoop         // while                                  // NONE                         //
	BreakStatement    // break                                  // NONE                         //
	ContinueStatement // continue                               // NONE                         //
	IfStatement       // if                                     // NONE                         //
	ElifStatement     // elif                                   // INVALID                      //
	ElseStatement     // else                                   // NONE                         //
	ComparisonSyntax  // ==, !=, >, <, >=, <=                   // SAME                         //
	And               // and                                    // NONE                         //
	Or                // or                                     // NONE                         //
	ListSyntax        // ,                                      // INVALID                      //
	Import            // import                                 // std, myFancyLibrary          //
	Dereference       // ^                                      // NONE                         //
	Comment           // # My comment 2                         // INVALID                      //
	Newline           // \n                                     // INVALID                      //
)

// Useful unicode charecters
const horizontalLine rune = '─'
const verticalLine rune = '│'
const crossingLines rune = '┼'
const upTriad rune = '┴'
const downTriad rune = '┬'
const rightTriad rune = '├'
const leftTriad rune = '┤'
const topLeftQuarterCircle rune = '╭'
const topRightQuarterCircle rune = '╮'
const bottomLeftQuarterCircle rune = '╰'
const bottomRightQuarterCircle rune = '╯'

// Useful ANIS codes
var ansiReset string = "\033[0m"
var ansiBold string = "\033[1m"

// Is either `Info`, `Warn`, or `Error` depending on the severity of a log
type logLevel uint8

const (
	Info logLevel = iota
	Warn
	Error
)

func checkRegisterListsAreTheSame(expectedRegisters []int, givenRegisters []registerAndLocation) codeParsingError {
	if len(expectedRegisters) != len(givenRegisters) {
		return codeParsingError{
			location: givenRegisters[0].location,
			msg:      errors.New("Expected " + fmt.Sprint(len(expectedRegisters)) + " items, got " + fmt.Sprint(len(givenRegisters)) + " items"),
		}
	}
	for i := range expectedRegisters {
		if expectedRegisters[i] != givenRegisters[i].register {
			return codeParsingError{
				location: givenRegisters[i].location,
				msg: errors.New("On register number " + fmt.Sprint(i+1) + ": expected the register r" +
					fmt.Sprint(expectedRegisters[i]) + ", got the register r" +
					fmt.Sprint(givenRegisters[i].register)),
			}
		}
	}
	return codeParsingError{}
}

func mapList[inputType any, outputType any](list []inputType, mapFunc func(inputType) outputType) []outputType {
	returnList := make([]outputType, len(list))
	for index, item := range list {
		returnList[index] = mapFunc(item)
	}
	return returnList
}

func mapListWithExtraArgs[inputType any, outputType any, extraArgsType any](list []inputType, mapFunc func(inputType, ...extraArgsType) outputType, extraArgs extraArgsType) []outputType {
	returnList := make([]outputType, len(list))
	for index, item := range list {
		returnList[index] = mapFunc(item, extraArgs)
	}
	return returnList
}

// Prints each error in `errors` with the 10 lines of code around where the
// error occured. Assumes that `errors` is in order of their `location.line`
// property.
func printErrorsInCode(fileName string, fileLines []string, errors []codeParsingError) bool {
	if len(errors) == 0 {
		return false
	}
	println(ansiBold, "===============", len(errors), "errors encountered in", fileName, "===============", ansiReset)
	charectersNeededForLineNumber := len(fmt.Sprint(errors[len(errors)-1].location.line))
	currentErrorIndex := 0
	shouldContinue := true
	for shouldContinue {
		lineNumber := max(0, errors[currentErrorIndex].location.line-5)
		groupEnd := min(len(fileLines), errors[currentErrorIndex].location.line+5)
		if currentErrorIndex != 0 {
			println("...")
		}
		for lineNumber < groupEnd {
			// TODO: Add code hightlighting to the line priting
			println(
				addWhitespaceToStart(fmt.Sprint(lineNumber+1), charectersNeededForLineNumber+1) +
					string(verticalLine) +
					fileLines[lineNumber])
			// For each error on the current line, print the error
			for errors[currentErrorIndex].location.line == lineNumber+1 {
				groupEnd = min(len(fileLines), errors[currentErrorIndex].location.line+5)
				print(strings.Repeat(" ", charectersNeededForLineNumber+2))
				for index, char := range fileLines[lineNumber] {
					if index >= errors[currentErrorIndex].location.column-1 {
						break
					} else if char == '\t' {
						print("\t")
					} else {
						print(" ")
					}
				}
				println("^ " + ansiBold + errors[currentErrorIndex].msg.Error() + ansiReset)
				if currentErrorIndex >= len(errors)-1 {
					shouldContinue = false
					break
				} else {
					currentErrorIndex++
				}
			}
			lineNumber++
		}
	}
	return true
}

func printTableSymbolsRow(
	leftSymbol rune,
	cellSymbol rune,
	cellSeperatorSymbol rune,
	rightSymbol rune,
	columnWidths ...int,
) {
	print(string(leftSymbol))
	for i, columnWidth := range columnWidths {
		if i > 0 {
			print(string(cellSeperatorSymbol))
		}
		for i := 0; i < columnWidth+2; i++ {
			print(string(cellSymbol))
		}
	}
	println(string(rightSymbol))
}

type tableCell struct {
	contents     string
	width        int
	rightAligned bool
}

func printTableRow(row []tableCell) {
	print(string(verticalLine) + " ")
	for i, cell := range row {
		if i > 0 {
			print(" " + string(verticalLine) + " ")
		}
		if cell.rightAligned {
			print(addWhitespaceToStart(cell.contents, cell.width))
		} else {
			print(addWhitespaceToEnd(cell.contents, cell.width))
		}
	}
	println(" " + string(verticalLine))
}

func addWhitespaceToEnd(input string, minimumChars int) string {
	if len(input) <= int(minimumChars) {
		return input + strings.Repeat(" ", int(minimumChars)-len(input))
	}
	return input
}

func addWhitespaceToStart(input string, minimumChars int) string {
	if len(input) <= int(minimumChars) {
		return strings.Repeat(" ", int(minimumChars)-len(input)) + input
	}
	return input
}

type textLocation struct {
	// Line and column indexing start at 1
	line   int
	column int
}

func splitSlice[T any](sliceToSplit []T, shouldSplit func(T) bool) [][]T {
	lastSplitIndex := -1
	returnValue := [][]T{}
	for index, item := range sliceToSplit {
		if shouldSplit(item) {
			add(&returnValue, sliceToSplit[lastSplitIndex+1:index])
			lastSplitIndex = index
		} else if index == len(sliceToSplit)-1 {
			add(&returnValue, sliceToSplit[lastSplitIndex+1:])
		}
	}
	return returnValue
}

func assert(err error) {
	if err != nil {
		panic("Unexpected internal state: Expected " + err.Error() + " to be true, but it was not.")
	}
}

func or(err1 error, err2 error) error {
	if err1 == nil || err2 == nil {
		return nil
	}
	return errors.New("(" + err1.Error() + ") || (" + err2.Error() + ")")
}

func eq[t comparable](value1 t, value2 t) error {
	if value1 == value2 {
		return nil
	}
	return errors.New("`" + fmt.Sprint(value1) + "` == `" + fmt.Sprint(value2) + "`")
}

func notEq[t comparable](value1 t, value2 t) error {
	if value1 != value2 {
		return nil
	}
	return errors.New("`" + fmt.Sprint(value1) + "` != `" + fmt.Sprint(value2) + "`")
}

func greaterThan[t int | int64 | float64](value1 t, value2 t) error {
	if value1 > value2 {
		return nil
	}
	return errors.New("`" + fmt.Sprint(value1) + "` > `" + fmt.Sprint(value2) + "`")
}

func lessThan[t int | int64 | float64](value1 t, value2 t) error {
	if value1 < value2 {
		return nil
	}
	return errors.New("`" + fmt.Sprint(value1) + "` < `" + fmt.Sprint(value2) + "`")
}

// Logs `msg` to stdout with the time that it was logged, and it's logLevel.
func log[T any](level logLevel, msg T) {
	print(time.Now().Format("15:04:05"))
	switch level {
	case Info:
		print("Info:  ")
	case Warn:
		print("Warn:  ")
	case Error:
		print("Error: ")
	}
	print(msg)
}

// Stores a list and a position within it
type listIterator[T any] struct {
	currentIndex int
	list         []T
}

func (list *listIterator[T]) next() bool {
	if list.currentIndex+1 < len(list.list) {
		list.currentIndex++
		return true
	}
	return false
}

func (list *listIterator[T]) get() *T {
	return &list.list[list.currentIndex]
}

// Appends to the end of a slice.
func add[T any](slice *[]T, itemToAppend ...T) {
	*slice = append(*slice, itemToAppend...)
}

// Inserts at the beginning of a slice.
func insert[T any](slice *[]T, itemToInsert T) {
	*slice = append([]T{itemToInsert}, *slice...)
}
