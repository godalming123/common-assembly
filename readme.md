# Common Assembly

> [!WARNING]
> Common assembly is pre-alpha, the (probably buggy) code needs at least some refactoring, and the compiler can barely compile a hello world. Other then a compiler, there also isn't any other developer tooling such a syntax highlighting or an LSP. Here is a list of things that need doing before even a V0.1 release:
>
> - Suppport more compilation targets other then just linux x86-64
> - Use brackets to specefy order of operation in conditions
> - Be able to call other functions (currently they can be defined but they cannot be used)
> - A (very basic) cross-platform standard library
> - A code highlighter
> - Support for importing one file from another
> - Have break and continue statements for while loops
> - Fix the up arrow (^) that points to an error not pointing to the correct charecter when there are tabs before the charecter that it is meant to point to
> - Fix comparisons needing to be in a certain order for the compiler to generate valid assembly
> - A do while loop as well as the normal while loop

# More things to do

A list of some other things that need doing before a V1.0 release:

- A type system
- Same developer experience as [high level assembly](https://github.com/hmhamza/hla-high-level-assembly-examples/blob/master/1.%20sumInputs.hla)
- A way to enforce that a program follows some style guidelines by accesing the compiler, for example forcing a program to name variables following a certain convention
- An `assert` function that dumps the program state if a condition is not met
- A way to debug code
- Add switch statements
- Lots of developer tooling:
  - Compiler:
    - Fast
    - Clear error messages
    - Suppports lots of compilation targets:
      - WASM
      - X86-64
      - Arm64
      - RiscV??
      - JS??
    - Supports lots of "OS"s:
      - Web
      - Windows
      - Linux
      - Mac
      - Android??
      - iOS??
  - Code highlighting
  - Formatter:
    - Insert spaces where necersarry
    - Insert tabs where necersarry (when nested)
    - Insert newlines where necersarry
  - An LSP server:
    - Autocomplete
    - Shows any error messages in the code
    - Symbol documentation
    - Symbol rename
    - Refactor code into a seperate function
  - A way to generate docs based on code comments

# Performance

A list of ways to benchmark common assembly compared to other languages:

- N-body simulation
- Inexed access to a sequence of 12 integers
- Generate mandlebrot set portable bitmap file
- Calculations with arbritrary precesion arithmatic
- Allocate, treverse, and deallocate binary trees

A list of performance improvements compared to low level languages such as C/C++/Rust/Odin/Zig/Jai/D:

- There is NO implicit data copying to the stack when a function is called

# DX Improvements compared to regular assembly (only applies once V0.1 is released)

- The same code works on many platforms
- No need to call functions by manually using `goto` and knowing exactly which register(s) they modify. Instead, use functions that just know the number of registers that they are using, if each register is an argument, and if they can modify the register. Then, they automatically use the stack to jump back to the caller.
- Use other peoples code with clear namespaces
- Abstract syscalls with a set of functions in the standerd library that are OS agnostic
- Abstract jump statements with:
  - While loops:
    - Break and continue statements
  - Switch statements:
    - Forced exhaustive case matching
  - If/else statements
- Data can be declared at any point in the program, instead of having to go in the data section
- Cleanup the syntax of assembly:
  - Use `memory++` instead of `inc memory`
  - Use `memory *= memory` instead of `mul memory, memory`
  - Use `memory = 0` instead of `mov memory, 0` or `xor memory, memory`
  - Use `pointerToFirstElementInList[index]` instead of `[pointerToFirstElementInList + index]`

# Code style for the go code in this repo

- If you have more than 4 levels of indentation, then you need to refactor your code
