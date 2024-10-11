# Programming language design considerations

<table>
  <tr>
    <th colspan="2">Challanges</th>
    <th>Solutions</th>
  </tr>
  <tr>
    <th colspan="2">Correnctness - how bug free is the code?</th>
    <th>Define/disallow undefined behavior</th>
  </tr>
  <tr>
    <td colspan="2">Variable mutation during a function being ran</td>
    <td>Define (at the function defintion and the function call) which variables it mutates, and have const by default</td>
  </tr>
  <tr>
    <td rowspan="2">Unexpected/unhandled errors</td>
    <td>Null pointer dereferencing</td>
    <td>Use a type system that forces pointers to not be null by default</td>
  </tr>
  <tr>
    <td>Failed syscalls</td>
    <td>Force the programmer to handle any possible failure cases for syscalls that could fail</td>
  </tr>
  <tr>
    <td colspan="2">Comparing 2 values of different types - like a pointer and a number</td>
    <td>Use a type system that forces at compile time that this does not happen</td>
  </tr>
  <tr>
    <td colspan="2">Unhandled possibilities</td>
    <td>Force exhaustive case matching</td>
  </tr>
  <tr>
    <th colspan="2">Readability - How readable is the code?</th>
    <th rowspan="2">Concise, clear syntax. The order of text in the program should show the order of runtime execution. Each package is a directory made up of common assembly files, with common assembly files that are in the same directory automatically getting access to each others functions.</th>
  </tr>
  <tr>
    <th colspan="2">Modifiability - How easy is it to make changes to the code?</th>
  </tr>
  <tr>
    <th colspan="2">Performance - How fast is the code at runtime?</th>
    <th>Create a syntax that shows the programmer how their code is slow</th>
  </tr>
</table>

# Common assembly design montra

- Stop using compiler "magic" to ensure that the program is correct, and that the binary runs fast. Instead, create a syntax that shows how the program is ineffecient, and the programmer will optimize their code better then any machine could.
- Stop hiding undefined behavior behind "pure" functional abstractions that pretend errors never happen. Instead, make a syntax that defines the undefined behavior, and forces the programmer to handle it, then the programmer can be sure that the program is correct.
