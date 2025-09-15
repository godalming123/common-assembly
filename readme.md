# Common Assembly

> [!WARNING]
> Common assembly is pre-alpha, the (probably buggy) code needs at least some refactoring, and the compiler can barely compile a hello world. Other then a compiler, there also isn't any other developer tooling such a syntax highlighting or an LSP. Here is a list of things that need doing before even a V0.1 release:
>
> - Fix /= and *=
> - Rework parts of `compiler.go` so that it creates one a strongly-typed set of instructions that can be converted into every archicetecture with the minimal possible code
>   - Support more compilation targets other then just linux x86-64
>   - Fix the assembler warnings that say "no instruction mnemonic suffix given and no register operands; using default for `...'"
> - Add support for floats
> - A (very basic) cross-platform standard library:
>   - An arena implementation:
>     - Would be based on 5 operations:
>       - `allocateArena`
>       - `expandArena`
>       - `shrinkArena`
>       - `resetArena` - frees every item in an arena without freeing the arena itself
>       - `deallocateArena`
>     - There would be a main arena that works by expanding and shrinking the program break rather than requesting backing memory and freeing backing memory for a large set of contiguous pages
>     - Depending on the language design, the operations might not be named in the code
> - A code highlighter
> - Support for importing one file from another
> - While loops:
>   - A do while loop as well as the normal while loop
> - Functions:
>   - Stop the main function from always exiting the process when it returns as it could be called by another function, in which case it should jump to where it was called from instead
>   - Add support for functions having `any` as a register
>   - Add a macro system that stops that user from manually having to count how many characters there are in a string that the user wants to print
> - Add the option to drop a variable from outside scope in the scope of an if/elif/else block as long as the variable is dropped in every branch of the block
> - Add more options to the command other then just compiling the main.ca file in the current directory:
>   - Ability to specify log level to be less than it is currently (no logs are outputted), or more than it is currently (the keywords and AST are outputted)
>     - flags: l0, l1, l2...
>   - Ability to run the program after it has compiled if the compilation was successful
>     - command: run
>   - Ability to automatically recompile the program when the source files are edited
>     - flag: watch
>   - Ability to compile a main.ca file from another directory
>     - the command arg
>   - Commands:
>     - Run
>     - Compile
>     - Help
>   - Flags:
>     - watch
>     - log-level
> - Internal go code: Add actual error messages when the `assert` statements fail, rather then just a stack trace

# More things to do

A list of some other things that need doing before a V1.0 release:

- A type system
- Add safety garuntees:
  - No dereferencing of invalid pointers:
    - No use after free
    - No null pointer dereference
  - No double free
  - Unused objects are forced to be cleaned up, so that memory leaks are not possible:
    - Free pointers
    - Close files
  - Thread safe code
  - Ways to achieve this:
    - Borrow checker (like in [rust](https://doc.rust-lang.org/beta/rust-by-example/scope.html))
    - Linear types (like in [austral](https://austral-lang.org/linear-types))
    - Mutable value semantics (like in [hylo](https://github.com/hylo-lang/hylo))
    - Reference counting (like in [swift](https://docs.swift.org/swift-book/documentation/the-swift-programming-language/automaticreferencecounting/))
    - Garbage collection (like in [go](https://tip.golang.org/doc/gc-guide))
- The ability for the compiler to automatically run code at compile time if:
  - That code doesn't depend on any state that would change at runtime
  - The compiler can reproduce any side affects that the code creates at runtime
- The compiler can be certain about both of these characteristics by adding function definitions that describe:
  - Which syscalls are used by the function (and all the functions it calls)
  - If that function or any function that it calls use the `sysret` instruction
  - What side affects are caused by that function and all of the functions that it calls
- Macros:
  - Uses of macros:
    - Implementing language features without updating the compiler:
      - Function overloading -> comptime map
      - Type system -> comptime struct with a type as a comptime enum and a value
      - Memory safety garuntees -> a way to enforce that code follows some style guidelines
      - Automatically building a function to serialize and desirialize structs (like serde)
      - Things that need direct access to jump to be implemented as performantly as possible:
        - Switch statements
        - If statements
        - Loops
        - Defer and errdefer statements
    - Transforming data with code at compile time, EG:
      - Counting how many characters there are in a string at comptime before it is printed
      - Generating prime numbers
      - Generating domain specific code, like SQL migrations, or the react compiler
  - These could be functions that:
    - Take in:
      - Either:
        - An AST
        - A list of keywords
        - A stream of keywords after the macro invocation:
          - The macro would decide when to stop accepting new keywords and let the programming language handle the following keywords instead of the macro continuing to handle keywords
        - If macros used a list of keywords or an AST, then the macro contents would have to be wrapped in square brackets say, and that would make code ugly if it uses a lot of macros (EG a macro for if)
        - If macros used an AST, then:
          - The macro would not be able to use a different syntax to the main language
          - The AST for the main language would have to be backwards compatible
        - If macros used keywords as input, then they would have to be able to specify a way to format the keywords that are passed to the macro
      - A list of the other function and/or variable definitions
    - Return either:
      - An AST of common assembly code (ideally the compiler would enforce that the macro always outputs a valid AST)
      - A list of errors present in the AST argument
    - Like [rust](https://doc.rust-lang.org/book/ch19-06-macros.html)
  - These could be functions that run at compile time to return other functions to run at runtime:
    - The problem with this is that this abstraction is too high level for common assembly (as it is right now):
      - Code that runs at compile time should not have to do register allocation, since performance of compiletime code is not that important
        - If register allocation isn't implemented in the compiler, then this would require separate comptime registers to use when calling a function at compiletime
      - Code that runs at compile time should have access to a type system
  - These could be functions with a expand tag to make them generate code that gets put at there call site like [jai](https://jai.community/t/macros/140)
  - These could be called like `macro!(keywords)`
- Same developer experience as [high level assembly](https://github.com/hmhamza/hla-high-level-assembly-examples/blob/master/1.%20sumInputs.hla)
- A way to enforce that a program follows some style guidelines by accessing the compiler
  - For example forcing a program to name variables following a certain convention
  - This could be achieved with a macro that wraps the code that you want to enforce the convention for
- An `assert` function that dumps the program state if a condition is not met
- Add switch statements
- Add support for accessing the lower 32 bits of a 64 bit register if there is a performance benefit
- Lots of developer tooling:
  - Compiler:
    - Fast
    - Clear error messages:
      - Give error messages for unused code, instead of ignoring most errors that can occur in functions that never get called
      - A warning for unused variables
      - Add support for printing multiple parser or compiler errors each time the compiler is ran rathor then only printing one at a time
      - Instead of just pointing to the first character of a keyword, the errors should point to the whole keyword
    - A `watch` command to automatically hot reload when the code changes if there aren't any compiler errors
    - Debugging, or the ability to generate executables with good debug symbols that work with debuggers and hot reload togethor
    - Generates optimized executables (see [benchmarking common assembly](#benchmarking-common-assembly) for how we would compare this with other optimizers):
      - [Here](https://llvm.org/docs/Passes.html) is a list of LLVM's optimization passes
      - Is optimized from source code alone:
        - Sometimes: Propagate constants:
          - Constants include:
            - A global constant variable
            - The argument of a function if it is set to the same value everywhere that the function is called
          - Always propagate constants if the size of the inline assembly is the same or less as the size of the assembly needed to reference the constant
        - Always: Replace `startsWith(myString[index:], "ab")` with `myString[index] == 'a' && index+1 < len(myString) && myString[index+1] == 'b'`
        - Always: Simplify logic based on propagated constants, for example remove the code in an if block where the condition is `0 == 1`
        - Always: Move values that are recomputed in the same way every time a loop runs to be outside the loop given the loop runs on average more than once
        - Always: Remove unused functions
        - Always: If the same code is always ran directly before or after the same function is called, then that code can be moved into the function definition
        - Sometimes: Unzip code that branches and regathers several times with the same condition into 2 branchless paths
        - Always: Replace `c := 0; for a < b {c += a; a += 1}` with `c := max(b-a, 0) * (a+b-1) / 2`
      - Is optimized from source code and assembly code:
        - Always: Use registers instead of the heap or stack
        - Sometimes: Inline code, with the following aspects making code more likely to be inlined:
          - The size of the inlined code is small (would have to include optimizations ran on the inlined code)
          - The function is only used once
          - The inlined code would be ran a large number of times
          - The size of the function call code is large
        - Always: Tail call optimizations
        - Always: Optimize array math to use SIMD
        - Always: Switch from using the `call` and `ret` instructions to using `jmp` instructions for functions that are used once
      - Is optimized from assembly code alone:
        - Always: If a jump label is never jumped to, and the code above the jump label always goes somewhere else, then the code between the jump label and the next jump label can be removed
        - Always: For arm, optimize `a = b; a += c` into `a = b + c`
        - Always: Use `inc register` rather than `add 1, register`
        - Always: Replace branching if statements with just instructions in some cases:
          - `compare REG1, REG2; jump_not_equal a; move REG3, REG4; a: ...` -> `compare REG1, REG2; move_equal REG3, REG4; ...`
        - Always: Use left shifts and right shifts instead of dividing/multiplying by powers of 2
        - Always: If there is a jump statement that jumps to another jump statement, then modify the first jump statement to jump directly to where the second jump statement jumps to
    - Supports lots of compilation targets:
      - WASM
      - X86-64
      - Arm64
      - LLVM IR
      - GPU code (see [ways of running code on the GPU](#ways-of-running-code-on-the-gpu))
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
  - Formatter (works on either an AST, or a list of keywords):
    - Insert spaces where necersarry
    - Insert tabs where necersarry (when nested)
    - Insert newlines where necersarry
  - An LSP server:
    - Autocomplete
    - Shows any error messages in the code
    - Symbol documentation
    - Symbol rename
    - Symbol picker
    - Refactor code into a separate function
  - A way to generate docs based on code comments (works on an AST)

# Installation instructions for Windows

1. Install Ubuntu WSL with [this guide](https://learn.microsoft.com/en-us/windows/wsl/install). From this point onwards, every command should be ran in WSL.
2. Install go:
   ```sh
   sudo apt-get install golang
   ```
3. Clone the common assembly code:
   ```sh
   git clone https://github.com/godalming123/common-assembly.git
   cd common-assembly
   ```
4. Compile the go code:
   ```sh
   go generate
   go compile
   ```
5. Put your common assembly code in `./main.ca`
6. Compile the common assembly in `./main.ca`:
   ```sh
   ./main
   ```
7. Run the binary produced by the common assembly compiler:
   ```sh
   ./out
   ```

# Performance

## Benchmarking common assembly

- To benchmark common assembly compared to other languages, a program that runs a loop several hundred times could be used
- The loop could:
  - Use seeded randomness to pick a performance benchmark from the following:
    - N-body simulation
    - Indexed access to a sequence of 12 integers
    - Generate mandlebrot set portable bitmap file
    - Calculations with arbitrary precision arithmetic
    - Allocate, traverse, and deallocate binary trees
    - Sort a list
    - Search for prime numbers
    - Find a number's factorial
    - Decoding and encoding for JSON and TOML
    - Search for [Munchausen numbers](https://en.wikipedia.org/wiki/Perfect_digit-to-digit_invariant)
    - Run this code:
      ```go
      func testOptimizationUnoptimized(a int, b int) int {
          c := a + b
          for a < c {
              a += 1
              b -= a
          }
          return b
      }
      ```
      To test if a compiler can optimize it into this code:
      ```go
      func testOptimizationOptimized(a int, b int) {
          mult := 0
          if b > 0 {
              mult = b
          }
          return b - (mult * (a + (b+1)/2))
      }
      ```
  - Run the performance benchmark
  - Print the result of the performance benchmark to stdout

## Common assembly performance improvements

A list of performance improvements compared to low level languages such as C/C++/Rust/Odin/Zig/Jai/D:

- There is NO implicit data copying:
  - To the stack when a function is called
  - Of the data in an array when it is expanded beyond it's max capacity
  - Of an array when each of it's elements are iterated over
- The compiler knows about all of the allocations in a program, so it:
  - Can optimize multiple allocations into one allocation
  - Can reuse previously allocated data in new allocations, provided the previous data is not used again
- Pointers are unique by default, so the compiler can produce better optimizations by making more assumptions about the code
- All of the code is optimized in one compilation unit, so the optimizer knows exactly what the code in other files and libraries does, rather than having to guess
- The compiler, language, and tooling help the programmer to optimize their code:
  - There should be some representation of the optimized code that the compiler generates at it's lowest level that is still CPU architecture independent
  - There could be a language feature for each function to specify another function to generate it's fuzzing data. Tooling could use this fuzzing data to test the performance of a function by running the function with this fuzzing data, and creating a graph that compares the size of the function's different arguments with the time it takes to run, and it's peak memory consumption.
- A design that de-emphasizes or does not support polymorphism, since polymorphism:
  - Requires another layer of pointer indirection
  - Makes moving code from the function definition to the function call and vice-versa harder for the optimizer

# DX Improvements compared to regular assembly (only applies once V0.1 is released)

- The same code works on many platforms
- No need to call functions by manually using `call` and knowing exactly which register(s) they modify. Instead, use functions that just have a set of argument registers that the caller passes a value into for the callee to use, and a set of mutated registers that the callee can mutate to process and return data to the caller.
- Use other peoples code with clear namespaces
- Abstract syscalls with a set of functions in the standard library that are OS agnostic
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
- Lines cannot be longer than 80 characters
- Function calls can be formatted in three ways:
  - If the resulting line is <=80 characters long, the function call is put on one line
  - If the resulting lines are <= 80 characters long, and there are only 2 lines, the function call can be hard-wrapped by it's arguments
  - Otherwise, each argument in the function call is given it's own line

# Stargazers over time

[![Stargazers over time](https://starchart.cc/godalming123/common-assembly.svg)](https://starchart.cc/godalming123/common-assembly)

# Ways of running code on the GPU

> [!NOTE]
> These lists are non-exhaustive. Compiling common assembly to GPU code is barely a vague idea right now.

## Shader languages

All of the below are maintained, and can create compute shaders.

| Abbreviation                              | API(s)                  |
| ----------------------------------------- | ----------------------- |
| HLSL                                      | DirectX                 |
| GLSL                                      | Vulkan and OpenGL       |
| MSL                                       | Metal                   |
| WGSL                                      | WebGPU                  |
| Cuda                                      | Cuda and zluda          |
| ROCm                                      | ROCm                    |
| [OpenCL](https://www.khronos.org/opencl/) | -                       |
| [SPIR-V](https://www.khronos.org/spirv/)  | Vulkan                  |
| [SysCl](https://www.khronos.org/sycl/)    | AdaptiveCpp and TriSYCL |

## APIs

| Name                                                                  | Maintained | Open source | Linux | Windows | Mac | AMD | Intel | Nvidia |
| --------------------------------------------------------------------- | ---------- | ----------- | ----- | ------- | --- | --- | ----- | ------ |
| [Vulkan](https://www.vulkan.org/)                                     | ✓          | ✓           | ✓     | ✓       | ✓   | ✓   | ✓     | ✓      |
| [WebGPU](https://developer.mozilla.org/en-US/docs/Web/API/WebGPU_API) | ✓          | ✓           | ✓     | ✓       | ✓   | ✓   | ✓     | ✓      |
| [AdaptiveCpp](https://github.com/AdaptiveCpp/AdaptiveCpp)             | ✓          | ✓           | ✓     | ✓       | ✓   | ✓   | ✓     | ✓      |
| (TriSYCL)[https://github.com/triSYCL/triSYCL]                         | ✓          | ✓           | ✓     | ✓       | ✓   | ✓   | ✓     | ✓      |
| DirectX                                                               | ✓          | ✗           | ✗     | ✓       | ✗   | ✓   | ✓     | ✓      |
| [ROCm](https://rocm.docs.amd.com/)                                    | ✓          | ✓           | ✓     | ✗       | ✗   | ✓   | ✗     | ✗      |
| [ZLUDA](https://github.com/vosen/ZLUDA) (still in development)        | ✓          | ✓           | ✓     | ✓       | ✗   | ✓   | ✗     | ✗      |
| [CUDA](https://developer.nvidia.com/cuda-toolkit)                     | ✓          | ✗           | ✓     | ✓       | ✗   | ✗   | ✗     | ✓      |
| [Metal](https://developer.apple.com/metal/)                           | ✓          | ✗           | ✗     | ✗       | ✓   | ✗   | ✗     | ✗      |
| OpenGL                                                                | ✗          | ✓           | ✓     | ✓       | ✗   | ✓   | ✓     | ✓      |

## Transpiling shader languages

| From   | To              | Via                                |
| ------ | --------------- | ---------------------------------- |
| HLSL   | SPIR-V          | DirectX shader compiler or glslang |
| GLSL   | SPIR-V          | Glslang                            |
| SPIR-V | MSL, HLSL, GLSL | SPIR-V Cross                       |
