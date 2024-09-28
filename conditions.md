# Conditions

Comparisons consist of `==`, `!=`, `>=`, `>`, `<=`, or `<` inbetween 2 values. Comparisons on there own make valid conditions:

```
func isNaturalNumber(number int) bool {
  return number > 0
}
```

Comparisons with arrows can be chained as long as the arrows point in the same direction:

```
func listRangeIsValid(listLen uint, listRangeStart uint, listRangeEnd uint) bool {
  return 0 <= listRangeStart <= listRangeEnd < listLen
}
```

The only other comparison that can be chained is `==`:

```
func isEqual(a uint, b uint, c uint) bool {
  return a == b == c
}
```

Comparisons can be combined to make conditions using `and` and `or`:

```
func pointIsOnScreen(screenWidth uint, screenHeight uint, pointX uint, pointY uint) bool {
  return 0 <= pointX < screenWidth and 0 <= pointY < screenHeight
}
func charecterIsDigit(char byte) bool {
  return '0' <= char <= '9' or char == '.'
}
```

Just `true` or `false` also make valid conditions:

```
func doForever(myFunctionn func) {
  while true {
    myFunction()
  }
}
func doNever(myFunctionn func) {
  while false {
    myFunction()
  }
}
```

`!=` cannot be chained since if you have `a != b != c`, then it is not clear if the comparison evaluates to false when `a == c`:

```
func isDifferent(a bool, b bool, c bool) {
  return a != b and a != c and b != c
}
```

`and` is more important in order of operations then `or`:

```
func slowCompilationSpeed(slowComputer bool, lang string) bool {
  return slowComputer and lang == "Rust" or lang == "Cpp" or lang == "C++"
}
```

# Conditions that are hard to compile with performance equivalant to assembly

```
func doIf(conditon bool, functionToDo func) {
  if condition {
    functionToDo()
  }
}
func doIfNot(conditon bool, functionToDo func) {
  if !condition {
    functionToDo()
  }
}
```
