# Common Assembly

> [!WARNING]
> Common assembly is pre-alpha, the (probably buggy) code needs at least some refactoring, and the compiler can barely compile a hello world. Other then a compiler, there also isn't any other developer tooling such a syntax highlighting or an LSP. Here is a list of things that need doing before even a V0.1 release:
>
> - Fix /= and *=
> - Fix myFunction(0) cuasing an assertion to fail in the compiler
> - Rework parts of `compiler.go` so that it creates one a strongly-typed set of instructions that can be converted into every archicetecture with the minimal possible code
>   - Suppport more compilation targets other then just linux x86-64
>   - Fix the assembler warnings that say "no instruction mnemonic suffix given and no register operands; using default for `...'"
> - Add support for floats
> - A (very basic) cross-platform standard library
> - A code highlighter
> - Support for importing one file from another
> - While loops:
>   - A do while loop as well as the normal while loop
> - Functions:
>   - Stop the main function from always exiting the process when it returns as it could be called by another function, in which case it should jump to where it was called from instead
>   - Add support for functions having `any` as a register
>   - Add a macro system that stops that user from manually having to count how many charecters there are in a string that the user wants to print
> - Add the option to drop a variable from outside scope in the scope of an if/elif/else block as long as the variable is dropped in every branch of the block
> - Add more options to the command other then just compiling the main.ca file in the current directory:
>   - Ability to specify log level to be less than it is currently (no logs are outputted), or more than it is currently (the keywords and AST are outputted)
>   - Ability to run the program after it has compiled if the compilation was succesful
>   - Ability to automatically recompile the program when the source files are edited
>   - Ability to compile a main.ca file from another directory
> - Internal go code: Add actual error messages when the `assert` statements fail, rather then just a stack trace

# More things to do

A list of some other things that need doing before a V1.0 release:

- A type system
- Same developer experience as [high level assembly](https://github.com/hmhamza/hla-high-level-assembly-examples/blob/master/1.%20sumInputs.hla)
- A way to enforce that a program follows some style guidelines by accesing the compiler, for example forcing a program to name variables following a certain convention
- An `assert` function that dumps the program state if a condition is not met
- A way to debug code
- Add switch statements
- Add support for accessing the lower 32 bits of a 64 bit register if there is a performance benefit
- Lots of developer tooling:
  - Compiler:
    - Fast
    - Clear error messages:
      - Give error messages for unused code, instead of ignoring most errors that can occur in functions that never get called
      - A warning for unused variables
      - Add support for printing multiple parser or compiler errors each time the compiler is ran rathor then only printing one at a time
      - Instead of just pointing to the first charecter of a keyword, the errors could point to the whole keyword
    - Generates optimized executables:
      - Tail call optimizations
      - Use left shifts and right shifts instead of dividing/multiplying by binary numbers
      - If there is a jump statement that jumps to another jump statement, then modify the first jump statement to jump directly to where the second jump statement jumps to
      - Use `inc register` rather than `add 1, register`
      - If a jump label is never jumped to, and the code above the jump label always goes somewhere else, then the code between the jump label and the next jump label can be removed
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
    - Symbol picker
    - Refactor code into a seperate function
  - A way to generate docs based on code comments

# Performance

A list of ways to benchmark common assembly compared to other languages:

- N-body simulation
- Indexed access to a sequence of 12 integers
- Generate mandlebrot set portable bitmap file
- Calculations with arbritrary precesion arithmatic
- Allocate, treverse, and deallocate binary trees

A list of performance improvements compared to low level languages such as C/C++/Rust/Odin/Zig/Jai/D:

- There is NO implicit data copying to the stack when a function is called

# DX Improvements compared to regular assembly (only applies once V0.1 is released)

- The same code works on many platforms
- No need to call functions by manually using `call` and knowing exactly which register(s) they modify. Instead, use functions that just have a set of argument registers that the caller passes a value into for the callee to use, and a set of mutated registers that the callee can mutate to process and return data to the caller.
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
  - Use `memory += memory` instead of `add memory, memory`
  - Use `memory = 0` instead of `mov memory, 0` or `xor memory, memory`
  - Use `pointerToFirstElementInList[index]` instead of `[pointerToFirstElementInList + index]`

# Code style for the go code in this repo

- If you have more than 4 levels of indentation, then you need to refactor your code
- Lines cannot be longer than 100 charecters
