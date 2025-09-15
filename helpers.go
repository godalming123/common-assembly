package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
	//"github.com/davecgh/go-spew/spew"
)

// Useful unicode characters
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

// Useful ANSI codes
var ansiReset string = "\033[0m"
var ansiBold string = "\033[1m"

// Is either `Info`, `Warn`, or `Error` depending on the severity of a log
type logLevel uint8

const (
	Info logLevel = iota
	Warn
	Error
)

func checkRegisterListsAreTheSame(expectedRegisters []Register, givenRegisters []registerAndLocation) codeParsingError {
	if len(expectedRegisters) != len(givenRegisters) {
		return codeParsingError{
			textLocation: givenRegisters[0].location,
			msg:          errors.New("Expected " + fmt.Sprint(len(expectedRegisters)) + " items, got " + fmt.Sprint(len(givenRegisters)) + " items"),
		}
	}
	for i := range expectedRegisters {
		if expectedRegisters[i] != givenRegisters[i].register {
			return codeParsingError{
				textLocation: givenRegisters[i].location,
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

// Println cannot be passed as a function arg in go
func passablePrintln(args ...any) {
	fmt.Println(args...)
}

func codeToAssembly(code string, printLineFunc func(...any)) (string, []codeParsingError) {
	printLineFunc("Lexing into a list of keywords...")
	keywords, errs := lexCode(code)
	if len(errs) > 0 {
		return "", errs
	}

	printLineFunc("Parsing keywords into abstract syntax tree...")
	AST, err := parseTopLevelASTitems(keywords)
	if err.msg != nil {
		return "", []codeParsingError{err}
	}

	// TODO: Figure out the best method to print the AST type
	// spew.Dump(AST)

	printLineFunc("Compiling abstract syntax tree into assembly...")
	return compileAssembly(AST)
}

// Prints each error in `errors` with the 10 lines of code around where the
// error occurred. Assumes that `errors` is in order of their `location.line`
// property.
func printErrorsInCode(
	fileName string,
	fileLines []string,
	errors []codeParsingError,
	printLineFunc func(...any),
) bool {
	if len(errors) == 0 {
		return false
	}
	printLineFunc(ansiBold, "===============", len(errors), "errors encountered in", fileName, "===============", ansiReset)
	charactersNeededForLineNumber := len(fmt.Sprint(errors[len(errors)-1].textLocation.line))
	currentErrorIndex := 0
	shouldContinue := true
	for shouldContinue {
		lineNumber := max(0, errors[currentErrorIndex].textLocation.line-5)
		groupEnd := min(len(fileLines), errors[currentErrorIndex].textLocation.line+5)
		if currentErrorIndex != 0 {
			printLineFunc("...")
		}
		for lineNumber < groupEnd {
			// TODO: Add code highlighting to the line printing
			printLineFunc(
				addWhitespaceToStart(fmt.Sprint(lineNumber+1), charactersNeededForLineNumber+1),
				string(verticalLine),
				fileLines[lineNumber],
			)
			// For each error on the current line, print the error
			for max(1, errors[currentErrorIndex].textLocation.line) == lineNumber+1 {
				groupEnd = min(len(fileLines), errors[currentErrorIndex].textLocation.line+5)
				print(strings.Repeat(" ", charactersNeededForLineNumber+2))
				for index, char := range fileLines[lineNumber] {
					if index >= errors[currentErrorIndex].textLocation.column-1 {
						break
					} else if char == '\t' {
						print("\t")
					} else {
						print(" ")
					}
				}
				printLineFunc("^ " + ansiBold + errors[currentErrorIndex].msg.Error() + ansiReset)
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
	cellSeparatorSymbol rune,
	rightSymbol rune,
	columnWidths ...int,
) {
	print(string(leftSymbol))
	for i, columnWidth := range columnWidths {
		if i > 0 {
			print(string(cellSeparatorSymbol))
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

func (location textLocation) location() textLocation { return location }

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

// Increments `currentIndex` if it isn't the index of the last item. Returns true if
// `currentIndex` was incremented, and false otherwise.
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
