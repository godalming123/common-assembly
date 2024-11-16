> [!WARNING]
> The compiler still needs updating in order to compile some of the examples here.

# 1. Registers

Here is a table of all of the 32 names to refer to 16 registers in common asembly:

| 64 bit register | Lower 32 bits of the register |
| --------------- | ----------------------------- |
| b0              | s0                            |
| b1              | s1                            |
| b2              | s2                            |
| b3              | s3                            |
| b4              | s4                            |
| b5              | s5                            |
| b6              | s6                            |
| b7              | s7                            |
| b8              | s8                            |
| b9              | s9                            |
| b10             | s10                           |
| b11             | s11                           |
| b12             | s12                           |
| b13             | s13                           |
| b14             | s14                           |
| b15             | s15                           |

When the registers are named individually, it means that they are only being used for one function call, and the compiler enforces that they do not effect how any code outside that function call runs:

```
b0, b5 = print(b4="Text to print\n", b3=14)
```

# 2. Variables

If you want to save the value in a register for more then one function call, then you have to reserve the register with a specefic variable. To do this, add the variable name next to a place in the code where the register is mutated:

```
b0 returnCode, b5 = print(b4="Testing variables\n", b3=18)
if returnCode != 0 {
  ... # Print call failed
}
```

Registers that are reserved for use with a specefic variable are used by naming the variable alone, and cannot be used by naming the register:

```
b0 returnCode, b5 = print(b4="Testing variables\n", b3=18)
b0 = exit(b5=returnCode)
# This code is invalid since b0 is reserved for the `returnCode` variable, so b0 cannot be mutated by naming the register since that would implicitly change the `returnCode` variable
```

To get round this, the `drop` keyword is used at the last place where a variable is used to free a register from being reserved for a variable:

```diff
b0 returnCode, b5 = print(b4="Testing variables\n", b3=18)
-b0 = exit(b5=returnCode)
+b0 = exit(b5=drop returnCode)
# Now the b0 register can be used just by naming the register, and `returnCode` is no longer a variable
```

A variable can be accessed on any line of code between where the variable is defined and where the variable is dropped, or falls out of scope. This isn't always the same as any point in time between when the variable is defined, and when the variable is dropped. This means that if a variable defined outside a while loop is dropped inside the while loop, then the variable can still be accessed on the second iteration of the loop as long as it is accessed above the `drop` statement.

```
b0 iteration = 0
while iteration < 10 {
  # `iteration` can still be accessed here, even on the second iteration of the while loop
  drop iteration++
  # `iteration` cannot be accessed here
  ...
}
```

# 3. Conditions

Comparisons consist of `==`, `!=`, `>=`, `>`, `<=`, or `<` in between 2 values. Comparisons on there own make valid conditions:

```
fn b0 greaterThan0 = isGreaterThan0(b0=number) {
  if number > 0 {
    return 1
  }
  return 0
}
```

Comparisons with arrows can be chained as long as the arrows point in the same direction:

```
fn b0 valid = listRangeIsValid(b0=listLen, b1=listRangeStart, b2=listRangeEnd) {
  if 0 <= listRangeStart <= listRangeEnd < listLen {
    return 1
  }
  return 0
}
```

The only other comparison that can be chained is `==`:

```
fn b0 equal = isEqual(b0=a, b1=b, b2=c) {
  if a == b == c {
    return 1
  }
  return 0
}
```

Comparisons can be combined to make conditions using `and` and `or`:

```
fn b0 onScreen = pointIsOnScreen(b0=screenWidth, b1=screenHeight, b2=pointX, b3=pointY) {
  if 0 <= pointX < screenWidth and 0 <= pointY < screenHeight {
    return 1
  }
  return 0
}

fn b0 digit = charecterIsDigit(b0=char) {
  if '0' <= char <= '9' or char == '.' {
    return 1
  }
  return 0
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
func b0 different = isDifferent(b0=a, b1=b, b2=c) {
  if a != b and a != c and b != c {
    return 1
  }
  return 0
}
```

`and` is more important in order of operations then `or`:

```
func b0 slow = slowCompilationSpeed(b0=slowComputer, b1=lang) {
  if slowComputer and lang == "Rust" or lang == "Cpp" or lang == "C++" {
    return 1
  }
  return 0
}
```

# 4. Functions

TODO: Create better docs then just some examples.

```
fn b0 result = double(b0=number) {
  number *= 2
  return number
}
```

```
fn b0 prime = isPrime(b1=num) {
  b0 factorToCheck = 2
  while factorToCheck < num {
    if (num % factorToCheck) == 0 {
      return 0
    }
    factorToCheck++
  }
  return 1
}
```

```
fn b0 result, b1 = pow(b2=base, b1=power) {
	b0 result = base
	while power > 1 {
		power--
		result *= base
	}
	return result
}
```

# 5. Syscalls

Common assembly provides the following syscall functions:

- `b0 exitCode = sysRead (b5=fileDescriptor, b4=buffer, b3=numberOfCharecters)`
- `b0 exitCode = sysWrite (b5=fileDescriptor, b4=text, b3=numberOfCharecters)`
- `b0 fileDescriptor = sysOpen (b5=fileName, b4=flags, b3=mode)`
- `b0 exitCode = sysClose (b5=fileDescriptor)`
- `b0 exitCode = sysBrk (b5=newBreak)`
- `b0 exitCode = sysExit (b5=status)`

These get compiled into inline assembly, for example `b0=sysWrite(b4="Hello world\n", b3=12, b5=1)` gets compiled to the following assembly for x86-64 linux:

```asm
text: .ascii "Hello world\n"
mov $1, %rax
mov $1, %rdi
mov $text, %rsi
mov %12, %rdx
syscall
```
