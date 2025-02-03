package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	fileName := "main.ca"

	fmt.Println("Reading the text in " + fileName + "...")
	rawText, err := os.ReadFile(fileName)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	assembly, errs := codeToAssembly(string(rawText), passablePrintln)
	if printErrorsInCode(fileName, strings.Split(string(rawText), "\n"), errs, passablePrintln) {
		os.Exit(1)
	}

	fmt.Println("Writing assembly to out.asm...")
	err = os.WriteFile("out.asm", []byte(assembly), 0644)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Assembling assembly to out.o...")
	out, err := exec.Command("as", "out.asm", "-o", "out.o").CombinedOutput()
	print(string(out))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Linking out.o to out...")
	out, err = exec.Command("ld", "out.o", "-o", "out").CombinedOutput()
	print(string(out))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
