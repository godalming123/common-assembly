> [!NOTE]
> Some of the examples in these docs use the `print` function, which is defined like so:
>
> ```
> r0 exitCode, r5 = print(r4=textToPrint, r3=numberOfCharecters) {
>   return r0=sysWrite(r5=1, textToPrint, numberOfCharecters)
> }
> ```

# 1. Registers

Here is a list of the 64-bit registers in common assembly:

- r0
- r1
- r2
- r3
- r4
- r5
- r6
- r7
- r8
- r9
- r10
- r11
- r12
- r13

These registers can be used to store data, but must be reset before the next function is called or returned since they modify the stack:

- r14
- r15

When the registers are named individually, it means that they are only being used for one function call, and the compiler enforces that they do not effect how any code outside that function call runs:

```
r0, r5 = print(r4="Text to print\n", r3=14)
```

> [!NOTE]
> **Only applies once common assembly can compile to several different architectures:**
> Different architectures have different numbers of registers. This means that on architectures with less registers, the later registers are actually just pointers to values stored in the data section, meaning that they are much slower than normal registers.

# 2. Variables

If you want to save the value in a register for more then one function call, then you have to reserve the register with a specefic variable. To do this, add the variable name next to a place in the code where the register is mutated:

```
r0 returnCode, r5 = print(r4="Testing variables\n", r3=18)
if returnCode != 0 {
  ... # Print call failed
}
```

Registers that are reserved for use with a specefic variable are used by naming the variable alone, and cannot be used by naming the register:

```
r0 returnCode, r5 = print(r4="Testing variables\n", r3=18)
r0 = sysExit(r5=returnCode)
```

The above code is invalid since r0 is reserved for the `returnCode` variable, so r0 cannot be mutated by naming the register since that would implicitly change the `returnCode` variable. To get round this, the `drop` keyword is used at the last place where a variable is used to free a register from being reserved for a variable:

```diff
r0 returnCode, r5 = print(r4="Testing variables\n", r3=18)
-r0 = sysExit(r5=returnCode)
+r0 = sysExit(r5=drop returnCode)
# Now the r0 register can be used here just by naming the register, and `returnCode` is no longer a variable.
```

Variables are implicitly dropped when they fall out of scope. A variable can be accessed on any line of code between where the variable is defined and where the variable is dropped. This is always the same as any point in time between when the variable is defined, and when the variable is dropped, since variables must be dropped in the scope that they were defined at. TODO: Although, in the future, the ability to drop a variable defined from outside scope in an if/elif/else statement may be added, as long as the variable is dropped in all of the branches of the statement.

# 3. Unimplemented: Data types (TODO: implement this)

Data types define what is meant by the 64 bits of data that a register stores. Here is a list of data types in common assembly:

- int
- uint
- float
- pointer(typeItPointsTo)
- list(typeOfListItem) - a pointer to the first value and a uint for the number of items in the list
- enum:
  - In the final type system, I would want this to have a value that can change in type
  - This would be the type used for booleans

# 4. Operations

Here is a table of the x86-64 assembly instructions generated for the given operations:

<table>
  <tr>
    <th rowspan="2">Operation</th>
    <th colspan="7">Instruction generated for registers of type</th>
  </tr>
  <tr>
    <th>Float</th>
    <th>Int</th>
    <th>Uint</th>
    <th>Pointer</th>
  </tr>
  <tr>
    <td>+=</td>
    <td>add</td>
    <td>iadd</td>
    <td>iadd</td>
    <td>invalid operation</td>
  </tr>
  <tr>
    <td>-=</td>
    <td>sub</td>
    <td>isub</td>
    <td>isub</td>
    <td>invalid operation</td>
  </tr>
  <tr>
    <td>*=</td>
    <td>TODO</td>
    <td>TODO</td>
    <td>TODO</td>
    <td>invalid operation</td>
  </tr>
  <tr>
    <td>/=</td>
    <td>TODO</td>
    <td>TODO</td>
    <td>TODO</td>
    <td>invalid operation</td>
  </tr>
</table>

# 5. Conditions

Comparisons consist of `==`, `!=`, `>=`, `>`, `<=`, or `<` in between 2 values. Comparisons on there own make valid conditions:

```
fn r0 greaterThan0 = isGreaterThan0(r0=number) {
  if number > 0 {
    return r0=1
  }
  return r0=0
}
```

Comparisons with arrows can be chained as long as the arrows point in the same direction:

```
fn r0 valid = listRangeIsValid(r0=listLen, r1=listRangeStart, r2=listRangeEnd) {
  if 0 <= listRangeStart <= listRangeEnd < listLen {
    return r0=1
  }
  return r0=0
}
```

The only other comparison that can be chained is `==`:

```
fn r0 equal = isEqual(r0=a, r1=b, r2=c) {
  if a == b == c {
    return r0=1
  }
  return r0=0
}
```

Comparisons can be combined to make conditions using `and` and `or`:

```
fn r0 onScreen = pointIsOnScreen(r0=screenWidth, r1=screenHeight, r2=pointX, r3=pointY) {
  if 0 <= pointX < screenWidth and 0 <= pointY < screenHeight {
    return r0=1
  }
  return r0=0
}

fn r0 digit = charecterIsDigit(r0=char) {
  if '0' <= char <= '9' or char == '.' {
    return r0=1
  }
  return r0=0
}
```

Just `true` or `false` also make valid conditions:

```
while true {
  # This code is ran until `break`
}
while false {
  # Thid code is never ran
}
```

`!=` cannot be chained since if you have `a != b != c`, then it is not clear if the comparison evaluates to false when `a == c`:

```
func r0 different = isDifferent(r0=a, r1=b, r2=c) {
  if a != b and a != c and b != c {
    return r0=1
  }
  return r0=0
}
```

`and` is more important in order of operations then `or`:

```
// TODO: This example does not work with the current common assembly compiler
func r0 slow = slowCompilationSpeed(r0=slowComputer, r1=lang) {
  if slowComputer and lang == "Rust" or lang == "Cpp" or lang == "C++" {
    return r0=1
  }
  return r0=0
}
```

# 6. Functions

TODO: Create better docs than just some examples.

```
fn r0 result = double(r0=number) {
  number *= 2
  return number
}

r0 number = getNumber()
number = double(number)

r0 number = double(r0=7)

r1 number = 5
r0 doubled = double(r0=number)
```

```
fn r0 prime = isPrime(r1=num) {
  r0 factorToCheck = 2
  while factorToCheck < num {
    if (num % factorToCheck) == 0 {
      return r0=0
    }
    factorToCheck++
  }
  return r0=1
}
```

```
fn r0 result, r1 = pow(r2=base, r1=power) {
	r0 result = base
	while power > 1 {
		power--
		result *= base
	}
	return result
}
```

# 7. Syscalls

Common assembly provides the following syscall functions:

- `r0 exitCode: i64 = sysRead (r5=fileDescriptor: i64, r4=buffer: pointer, r3=numberOfCharecters: i64)`
- `r0 exitCode: i64 = sysWrite (r5=fileDescriptor: i64, r4=text: i64, r3=numberOfCharecters: i64)`
- `r0 fileDescriptor: i64 = sysOpen (r5=fileName: pointer, r4=flags: i64, r3=mode: i64)`
- `r0 exitCode: i64 = sysClose (r5=fileDescriptor: i64)`
- `r0 exitCode: i64 = sysBrk (r5=newBreak: i64)`
- `r0 exitCode: i64 = sysExit (r5=status: i64)`

These get compiled into inline assembly, for example `r0=sysWrite(r4="Hello world\n", r3=12, r5=1)` gets compiled to the following assembly for x86-64 linux:

```asm
text: .ascii "Hello world\n"
mov $1, %rax
mov $1, %rdi
mov $text, %rsi
mov %12, %rdx
syscall
```

# 8. TODO: Modules and imports

- Modules would be defined by creating a file with the `.mod` file extension in the root directory of the module
  - Then, any files within that directory or any subdirectories would be part of that module
- Modules would follow a flat structure - they cannot be nested
- There would be 3 options for how much the definition can be accessed without using `import`:
  - Can only be accessed from the same file (the default)
  - Can only be accessed from files that are within the same directory as the definition
  - Can be accessed from any file that is both:
    - Within the same directory as the definition or a parent directory of the definition
    - Within the same module as the definition
- Definitions from other modules would be accessable under a prefix if they are imported in the file that is accessing the definition and marked as `public` at the definition
- If a definition has unnecersarry permissions, the compiler would show a warning
