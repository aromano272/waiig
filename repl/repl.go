package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"waiig/evaluator"
	"waiig/lexer"
	"waiig/object"
	"waiig/parser"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	env := object.NewEnvironment()

	parseStd(env)

	for {
		fmt.Print(PROMPT)
		scanned := scanner.Scan()

		if !scanned {
			return
		}

		line := scanner.Text()

		l := lexer.New(line)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		evaluated := evaluator.Eval(program, env)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}

func parseStd(env *object.Environment) {
	data, err := os.ReadFile("std/std.monkey")
	if err != nil {
		panic(err)
	}

	input := string(data)

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	evaluator.Eval(program, env)
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
