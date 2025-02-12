package main

import (
	_ "embed"
	"strings"
	"testing"
)

func TestMainCode(t *testing.T) {
	testOrBenchmarkMainCode(t)
}

func BenchmarkMainCode(b *testing.B) {
	testOrBenchmarkMainCode(b)
}

func testOrBenchmarkMainCode(tb testing.TB) {
	assembly, errs := codeToAssembly(mainCommonAssemblyCode, tb.Log)
	if printErrorsInCode("main.ca", strings.Split(mainCommonAssemblyCode, "\n"), errs, tb.Log) {
		tb.FailNow()
	}
	if assembly != mainExpectedAssemblyCode {
		// TODO: Ideally this would print the difference between the expected
		// assembly, and the generated assembly, rathor than just printing the
		// generated assembly.
		tb.Fatalf("Got the wrong assembly. Here is the assembly `codeToAssmbly` returned:\n%s", assembly)
	}
}

func TestInvalidFunctionArgs(t *testing.T) {
	code := `
		fn r0, r5, r4, r3 = main() {
			r0 = sysWrite(0) # Just 0 is not a function argument
		}
	`
	_, errs := codeToAssembly(code, t.Log)
	if len(errs) == 0 {
		t.Fatal("The compiler somehow thinks that the invalid code is valid")
	}
	if len(errs) > 1 {
		t.Log("Expected invalid function args test code to only give one error")
		printErrorsInCode("test code", strings.Split(code, "\n"), errs, t.Log)
		t.FailNow()
	}
	if errs[0].line != 3 || errs[0].column != 18 {
		t.Fatal("Expected invalid function args test code to give an error at line 3 and column 18, but it gave an error at line", errs[0].line, "and column", errs[0].column)
	}
}

//go:embed main.ca
var mainCommonAssemblyCode string
var mainExpectedAssemblyCode = `.global _start
.text
dataSectionLabel1: .ascii "Enter your name: "
dataSectionLabel2: .ascii "You entered: "
dataSectionLabel3: .ascii "\nCounting from 0 to 9...\n"
dataSectionLabel4: .ascii "\n"
dataSectionLabel5: .ascii "Point is not on the screen\n"
dataSectionLabel6: .ascii "Point is on the screen\n"
_start:
mov $1, %rdi
mov $dataSectionLabel1, %rsi
mov $17, %rdx
mov $1, %rax
syscall
mov $0, %rdi
mov $12, %rax
syscall
mov %rax, %r15
mov %rax, %r14
mov %rax, %r10
jmp jumpLabel2
jumpLabel1:
cmp %r14, %r10
jg jumpLabel4
add $4096, %r10
mov %r10, %rdi
mov $12, %rax
syscall
jumpLabel4:
mov $0, %rdi
mov %r14, %rsi
mov $1, %rdx
mov $0, %rax
syscall
cmp $0, %rax
jge jumpLabel5
mov %rax, %rdi
mov $60, %rax
syscall
jmp jumpLabel6
jumpLabel5:
cmp $0, %rax
je jumpLabel8
cmp $'\n', (%r14)
jne jumpLabel7
jumpLabel8:
jmp jumpLabel3
jumpLabel7:
jumpLabel6:
add $8, %r14
jumpLabel2:
jmp jumpLabel1
jumpLabel3:
mov $1, %rdi
mov $dataSectionLabel2, %rsi
mov $13, %rdx
mov $1, %rax
syscall
mov %r14, %rdx
sub %r15, %rdx
mov $1, %rdi
mov %r15, %rsi
mov $1, %rax
syscall
mov %r15, %rdi
add $4096, %rdi
mov $12, %rax
syscall
mov $1, %rdi
mov $dataSectionLabel3, %rsi
mov $25, %rdx
mov $1, %rax
syscall
mov %r15, %rsi
mov $'0', (%rsi)
jmp jumpLabel10
jumpLabel9:
mov $1, %rdi
mov $1, %rdx
mov $1, %rax
syscall
inc (%rsi)
mov $dataSectionLabel4, %rsi
mov $1, %rdi
mov $1, %rdx
mov $1, %rax
syscall
mov %r15, %rsi
cmp $'9', (%rsi)
jle jumpLabel12
jmp jumpLabel11
jumpLabel12:
jumpLabel10:
jmp jumpLabel9
jumpLabel11:
mov $300, %rax
mov $30, %rbx
mov $100, %rcx
mov $250, %rdx
mov $0, %rsi
jmp jumpLabel22
jumpLabel21:
cmp $0, %rax
jne jumpLabel19
mov $1, %rdi
mov $dataSectionLabel5, %rsi
mov $27, %rdx
mov $1, %rax
syscall
jmp jumpLabel20
jumpLabel19:
mov $1, %rdi
mov $dataSectionLabel6, %rsi
mov $23, %rdx
mov $1, %rax
syscall
jumpLabel20:
mov $60, %rax
mov $0, %rdi
syscall
jumpLabel22:
cmp $0, %rsi
jne jumpLabel14
cmp $0, %rax
jl jumpLabel13
cmp %rax, %rcx
jle jumpLabel13
jumpLabel16:
cmp $0, %rbx
jl jumpLabel13
cmp %rbx, %rdx
jle jumpLabel13
jumpLabel17:
jumpLabel15:
jumpLabel14:
mov $1, %rax
jmp jumpLabel21
jmp jumpLabel18
jumpLabel13:
mov $0, %rax
jmp jumpLabel21
jumpLabel18:
jmp jumpLabel21
`
