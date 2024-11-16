package main

import (
	"fmt"
	"strings"
	"time"
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

func assert(condition bool) {
	if !condition {
		// TODO: Say where the assertion is in the code
		panic("Unexpected internal state: Assert failed")
	}
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
