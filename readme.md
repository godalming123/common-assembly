# Common Assembly

> [!WARNING]
> Common assembly is pre-alpha, the (probably buggy) code needs at least some refactoring, and the compiler can barely compile a hello world. Other then a compiler, there also isn't any other developer tooling such a syntax highlighting or an LSP. Here is a list of things that need doing before even a V0.1 release:
>
> - Fix /= and *=
> - Rework parts of `compiler.go` so that it creates one a strongly-typed set of instructions that can be converted into every archicetecture with the minimal possible code
>   - Suppport more compilation targets other then just linux x86-64
>   - Fix the assembler warnings that say "no instruction mnemonic suffix given and no register operands; using default for `...'"
> - Add support for floats
> - A (very basic) cross-platform standard library
>   - An arena implementation that has 5 functions:
>     - `allocateArena`
>     - `expandArena`
>     - `shrinkArena`
>     - `resetArena` - frees every item in an arena without freeing the arena itself
>     - `deallocateArena`
>   - The main function would have an argument called `dataArena`, that works by expanding and shrinking the program break
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
- Add safety garuntees:
  - No dereferencing of invalid pointers:
    - No use after free
    - No null pointer dereference
  - No double free
  - Unused objects are forced to be cleaned up, so that memory leaks are not possible:
    - Free pointers
    - Close files
  - Thread safe code
- The ability for the compiler to automatically run code at compile time if:
  - That code doesn't depend on any state that would change at runtime
  - The compiler can reproduce any side affects that the code creates at runtime
- The compiler can be certain about both of these charecteristics by adding function definitions that describe:
  - Which syscalls are used by the function (and all the functions it calls)
  - If that function or any function that it calls use the `sysret` instruction
  - What side affects are caused by that function and all of the functions that it calls
- Macros
  - These could be functions that:
    - Take in:
      - An AST of code
      - A list of the other function definitions
    - Return either:
      - An AST of common assembly code
      - A list of errors present in the AST argument
  - These could be called like `macro!(AST)`
- Same developer experience as [high level assembly](https://github.com/hmhamza/hla-high-level-assembly-examples/blob/master/1.%20sumInputs.hla)
- A way to enforce that a program follows some style guidelines by accesing the compiler
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
      - Instead of just pointing to the first charecter of a keyword, the errors should point to the whole keyword
    - A `watch` command to automatically hot reload when the code changes if there aren't any compiler errors
    - Debugging, or the ability to generate executables with good debug symbols that work with debuggers and hot reload togethor
    - Generates optimized executables:
      - Tail call optimizations
      - Use left shifts and right shifts instead of dividing/multiplying by binary numbers
      - If there is a jump statement that jumps to another jump statement, then modify the first jump statement to jump directly to where the second jump statement jumps to
      - Use `inc register` rather than `add 1, register`
      - If a jump label is never jumped to, and the code above the jump label always goes somewhere else, then the code between the jump label and the next jump label can be removed
      - For arm, optimize `a = b; a += c` into `a = b + c`
    - Suppports lots of compilation targets:
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
    - Calculations with arbritrary precesion arithmatic
    - Allocate, traverse, and deallocate binary trees
    - Sort a list
    - Search for prime numbers
  - Run the performance benchmark
  - Print the result of the performace benchmark to stdout

## Common assembly performance improvements

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
- Lines cannot be longer than 80 charecters
- Function calls can be formatted in three ways:
  - If the resulting line is <=80 charecters long, the function call is put on one line
  - If the resulting lines are <= 80 charecters long, and there are only 2 lines, the function call can be hard-wrapped by it's arguments
  - Otherwise, each argument in the function call is given it's own line

# Stargazers over time

[![Stargazers over time](https://starchart.cc/godalming123/common-assembly.svg)](https://starchart.cc/godalming123/common-assembly)

# Ways of running code on the GPU

> [!NOTE]
> These lists are non-exhaustive. Compiling common assembly to GPU code is barely a vague idea right now.

## Shader languages

All of the below are maintained, and can create compute shaders.

| Abbreviation                              | API(s)            |
| ----------------------------------------- | ----------------- |
| HLSL                                      | DirectX           |
| GLSL                                      | Vulkan and OpenGL |
| MSL                                       | Metal             |
| WGSL                                      | WebGPU            |
| Cuda                                      | Cuda and zluda    |
| ROCm                                      | ROCm              |
| [OpenCL](https://www.khronos.org/opencl/) | -                 |
| [SPIR-V](https://www.khronos.org/spirv/)  | Vulkan            |
| [SysCl](https://www.khronos.org/sycl/)    | -                 |

## APIs

| Name                                                                  | Maintained | Open source | Linux | Windows | Mac | AMD | Intel | Nvidia |
| --------------------------------------------------------------------- | ---------- | ----------- | ----- | ------- | --- | --- | ----- | ------ |
| [Vulkan](https://www.vulkan.org/)                                     | ✓          | ✓           | ✓     | ✓       | ✓   | ✓   | ✓     | ✓      |
| [WebGPU](https://developer.mozilla.org/en-US/docs/Web/API/WebGPU_API) | ✓          | ✓           | ✓     | ✓       | ✓   | ✓   | ✓     | ✓      |
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
