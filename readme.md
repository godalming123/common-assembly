# Common Assembly

The speed of assembly with good DX, and 1 codebase for many platforms.

## Code example

```
# Prints the at `str` which is a pointer to a a list of charecters of length `strLen`
print :: (str arg rsi, strLen arg rdx, mem1 mut rax, mem2 mut rdi) {
	mem1 = 1
	mem2 = 1
	# TODO: Instead of `syscall`, for the user to say `syscall mut str, mut strLen, mut mem1, mut mem2` so that every possible mutation is explicit
	syscall
}

printLetter :: (char mutArg rsi, mem1 mut rdx, mem2 mut rax, mem3 mut rdi) {
  // TODO: Choose a syntx for switch statements
  if char == {
    'c', 'C':
      char = "We don't like C\n"
      mem1 = len .
    'a' <= exp <= 'z':
      char = "Char is a lower case letter.\n"
      mem1 = len .
    'A' <= exp <= 'Z':
      char = "Char is an uppercase letter.\n"
      mem1 = len .
    '0' <= exp <= '9':
      char = "Char is a number.\n"
      mem1 = len .
    else:
      char = "Char is unknown.\n"
      mem1 = len .
  }
  if char matches 'c', 'C' {
    char = "We don't like C\n"
    mem1 = len .
  } else if matches 'a' <= exp <= 'z' {
      char = "Char is a lower case letter.\n"
      mem1 = len .
  } else if mathces 'A' <= exp <= 'Z' {
    char = "Char is an uppercase letter.\n"
    mem1 = len .
  } else if matches '0' <= exp <= '9' {
      char = "Char is a number.\n"
      mem1 = len .
  } else {
    char = "Char is unknown.\n"
    mem1 = len .
  }
  print char, mem1, mut mem2, mut mem3
}
```

## Performance tests

- N-body simulation
- Inexed access to a sequence of 12 integers
- Generate mandlebrot set portable bitmap file
- Calculations with arbritrary precesion arithmatic
- Allocate, treverse, and deallocate binary trees

## TODO

- Same DX as [high level assembly](https://github.com/hmhamza/hla-high-level-assembly-examples/blob/master/1.%20sumInputs.hla)
- A way to enforce that a program follows some style guidelines by accesing the compiler, for example forcing a program to name variables following a certain convention
- Assert that dumps the program state on failure
- A way to debug code
- Support control flow syntax
- Handle jumping to the value at the current value on the stack at the end of a function
- Handle reseting the stack after a function call
- Handle integers, floats, and negatives
- A couple optimizations
- Support import

## DX Improvements compared to regular assembly

- The same code works on many platforms.
- No need to call functions by manually using `goto` and knowing exactly which register(s) they modify. Instead, use functions that just know the number of registers that they are using, if each register is an argument, and if they can modify the register. Then, they automatically use the stack to jump back to the caller.
- Use other peoples code with clear namespaces
- Abstract syscalls with a set of functions in the standerd library that are OS agnostic.
- Abstract jump statements with:
  - While loops:
    - Break and continue statements
  - Switch statements:
    - Forced exhaustive case matching
  - If/else statements
- Data can be declared at any point in the program, instead of having to go in the data section.
- Cleanup the syntax of assembly:
  - Use `memory++` instead of `add memory, 1`
  - Use `memory *= memory` instead of `mul memory, memory`
  - Use `memory = 0` instead of `mov memory, 0` or `xor memory, memory`
  - Use `pointerToFirstElementInList[index]` instead of `[pointerToFirstElementInList + index]`

## Performance improvements compared to C/C++/Rust/Odin/Zig/Jai/D

- There is NO data copying when a function is called
- If a variable allocated in main memory is no longer used, and then a different variable that takes up the same size in memory is created, then the new variable uses the memory from the old variable
- If a variable allocated in main memory is no longer used, and then a different variable that takes up less memory is created, then a small amount of memory is freed, and the new variable uses the memory from the old variable
- If a list has items removed from either end, then just the items are deallocated so the rest of the list uses the same portion of memory without any copying
- CANNOT HAPPEN WITHOUT FORCING FUNCTIONS TO DESCRIBE THEIR SIDE AFFECTS: Functions are automatically ran at compile time

## Potential improvements for the future

- (??) Make programs more correct by forcing type checking:
  - Some people say that [the type system should be designed around how a database works](https://spacetimedb.com/blog/databases-and-data-oriented-design)
  - [Varient types](https://ocaml.org/docs/basic-data-types#variants) are an intresting idea for modeling more complex types. **(How do lists work with varient types that can be diferent sizes in memory?)** However, one of go's language designers [would disagree](https://github.com/golang/go/issues/29649#issuecomment-454820179):
    > (...) Go intentionally has a weak type system, and there are many restrictions that can be expressed in other languages but cannot be expressed in Go. Go in general encourages programming by writing code rather than programming by writing types. (...)
  - Depending on how strict the type system is, [generics](https://go.dev/doc/tutorial/generics) might be necersarry to simplify functions so that (for example) there can be one `convertToInt` function rathor then severel for each different type of input
- (??) Force correct memory management
- (??) Add first class concurrency
- (??) Abstract the precession of numbers so they can be as precise as they need to be
- (??) Make combining data easier with formatted strings - `"hello ${name}!"` instead of `append("hello ", name, "!")`. Types would get implicitly cast when using formatted strings.
- (??) Force functions to describe their side affects:
  - Printing text to the screen
  - Randomness
  - Reading/writing files (EG: a directory compression program)
  - Getting the current time
  - Delibrately doing nothing for a period of time (EG: a timer)
- (??) Functional programming functions out of the box:
  - Map -> Runs a function that is parsed by an argument to transform each item in a list. (If the function returns nil for the item in the list, then it is not present in the returned list.)?
  - Flatten -> Flattens a 2D list into a 1D list such that [["Carrots", "Bananas", "Grapes"], ["Coffee", "Water", "Juice"], ["Apple", "Orange"]] becomes ["Carrots", "Bananas", "Grapes", "Coffee", "Water", "Juice", "Apple", "Orange"]
  - Sort :: fn(comparison: fn('a, 'b) -> bool) -> Sorts a list such that `comparison(list[n], list[n+1]) == true`
  - Reduce
  - FindUntil

## Tooling that needs to be good for a V1 release

### A compiler

Needs to be a good compiler:

- Fast
- Clear error messages

Needs to support lots of architectures:

- WASM
- X86-64
- Arm64
- RiscV??

And lots of OS's:

- Web
- Windows
- Linux
- Mac
- Android??
- iOS??

### An LSP

- Code highlighting
- Autocomplete
- Shows any error messages in the code
- Formatter:
  - Insert spaces where necersarry
  - Insert tabs where necersarry (when nested)
  - Insert newlines where necersarry
- Symbol documentation
- Symbol rename
- Refactor code into a seperate function

### Some other devtools

- A way to generate documentation based on code comments

# Code style

- If you have more than 4 levels of indentation, then you need to refactor your code
