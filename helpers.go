package main

import "time"

// Is either `Info`, `Warn`, or `Error` depending on the severity of a log
type logLevel uint8

const (
	Info logLevel = iota
	Warn
	Error
)

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

// Logs each error generated from parsing a block of code
//func logErrorsInText(text textAndPosition, errors []codeParsingError) {
//	codeParsingErrorIndex := 0
//	text.index = 0
//	text.line = 0
//	text.column = 0
//	for ()) {
//		if code[index] == '\n' {
//			// TODO: Print line numbers
//			line++
//			if line > (errors[codeParsingErrorIndex].line - 5) {
//				printingFile = true
//				if line == errors[codeParsingErrorIndex].line {
//					// TODO: Actually print the error messages
//				}
//			} else if line > (errors[codeParsingErrorIndex].line + 5) {
//				printingFile = false
//			}
//		}
//		if printingFile {
//			print(code[index])
//		}
//	}
//
//}

// Appends to the end of a slice.
func add[T any](slice *[]T, itemToAppend T) {
	*slice = append(*slice, itemToAppend)
}
