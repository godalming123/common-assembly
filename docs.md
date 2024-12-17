> [!WARNING]
> The compiler still needs updating in order to compile some of the examples here.

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
r0 = exit(r5=returnCode)
# This code is invalid since r0 is reserved for the `returnCode` variable, so r0 cannot be mutated by naming the register since that would implicitly change the `returnCode` variable
```

To get round this, the `drop` keyword is used at the last place where a variable is used to free a register from being reserved for a variable:

```diff
r0 returnCode, r5 = print(r4="Testing variables\n", r3=18)
-r0 = exit(r5=returnCode)
+r0 = exit(r5=drop returnCode)
# Now the r0 register can be used just by naming the register, and `returnCode` is no longer a variable
```

A variable can be accessed on any line of code between where the variable is defined and where the variable is dropped, or falls out of scope. This isn't always the same as any point in time between when the variable is defined, and when the variable is dropped. This means that if a variable defined outside a while loop is dropped inside the while loop, then the variable can still be accessed on the second iteration of the loop as long as it is accessed above the `drop` statement.

```
r0 iteration = 0
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
fn r0 greaterThan0 = isGreaterThan0(r0=number) {
  if number > 0 {
    return 1
  }
  return 0
}
```

Comparisons with arrows can be chained as long as the arrows point in the same direction:

```
fn r0 valid = listRangeIsValid(r0=listLen, r1=listRangeStart, r2=listRangeEnd) {
  if 0 <= listRangeStart <= listRangeEnd < listLen {
    return 1
  }
  return 0
}
```

The only other comparison that can be chained is `==`:

```
fn r0 equal = isEqual(r0=a, r1=b, r2=c) {
  if a == b == c {
    return 1
  }
  return 0
}
```

Comparisons can be combined to make conditions using `and` and `or`:

```
fn r0 onScreen = pointIsOnScreen(r0=screenWidth, r1=screenHeight, r2=pointX, r3=pointY) {
  if 0 <= pointX < screenWidth and 0 <= pointY < screenHeight {
    return 1
  }
  return 0
}

fn r0 digit = charecterIsDigit(r0=char) {
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
func r0 different = isDifferent(r0=a, r1=b, r2=c) {
  if a != b and a != c and b != c {
    return 1
  }
  return 0
}
```

`and` is more important in order of operations then `or`:

```
func r0 slow = slowCompilationSpeed(r0=slowComputer, r1=lang) {
  if slowComputer and lang == "Rust" or lang == "Cpp" or lang == "C++" {
    return 1
  }
  return 0
}
```

# 4. Functions

TODO: Create better docs then just some examples.

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
      return 0
    }
    factorToCheck++
  }
  return 1
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

# 5. Syscalls

Common assembly provides the following syscall functions:

- `r0 exitCode = sysRead (r5=fileDescriptor, r4=buffer, r3=numberOfCharecters)`
- `r0 exitCode = sysWrite (r5=fileDescriptor, r4=text, r3=numberOfCharecters)`
- `r0 fileDescriptor = sysOpen (r5=fileName, r4=flags, r3=mode)`
- `r0 exitCode = sysClose (r5=fileDescriptor)`
- `r0 exitCode = sysBrk (r5=newBreak)`
- `r0 exitCode = sysExit (r5=status)`

These get compiled into inline assembly, for example `r0=sysWrite(r4="Hello world\n", r3=12, r5=1)` gets compiled to the following assembly for x86-64 linux:

```asm
text: .ascii "Hello world\n"
mov $1, %rax
mov $1, %rdi
mov $text, %rsi
mov %12, %rdx
syscall
```
