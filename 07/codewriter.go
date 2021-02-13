package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type CodeWriter struct{
	commands []Command
	writer io.Writer
	name string
	labelCnt int
}

func NewCodeWriter(commands []Command, writer io.Writer, name string) *CodeWriter {
	return &CodeWriter{commands: commands, writer: writer, name: name, labelCnt: 0}
}

func (cw *CodeWriter)GenerateCode() error {
	for _, c := range cw.commands {
		switch c.CommandType {
		case CArithmetic:
			err := cw.writeArithmetic(c.command)
			if err != nil {
				return err
			}
		case CPush:
			err := cw.writePush(c.arg1, c.arg2)
			if err != nil {
				return err
			}
		case CPop:
			err := cw.writePop(c.arg1, c.arg2)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cw *CodeWriter)fPrintln(a string) {
	fmt.Fprintln(cw.writer, a)
}

func (cw *CodeWriter)writePush(segment string, index int) error {
	var code []string

	setIndex := func() []string {
		return []string{"@" + strconv.Itoa(index), "D=A"}
	}

	// load segment[index] into D register
	loadMemData := func(base string) []string {
		return append(setIndex(),  []string{"@" + base, "A=D+M", "D=M"}...)
	}

	switch segment {
	case "constant":
		code = setIndex()
	case "argument":
		code = loadMemData("ARG")
	case "local":
		code = loadMemData("LCL")
	case "this":
		code = loadMemData("THIS")
	case "that":
		code = loadMemData("THAT")
	case "pointer":
		code = append(setIndex(), []string{"@R3", "A=D+A", "D=M"}...)
	case "temp":
		code = append(setIndex(), []string{"@R5", "A=D+A", "D=M"}...)
	case "static":
		code = []string{"@" + cw.name + "." + strconv.Itoa(index), "D=M"}
	default:
		return fmt.Errorf("undefined segment: %s", segment)
	}
	code = append(code, []string{"@SP", "A=M", "M=D", "@SP", "M=M+1"}...)

	cw.fPrintln(strings.Join(code, "\n"))
	return nil
}

func (cw *CodeWriter)writePop(segment string, index int) error {
	var code []string

	pop := func(base string) []string {
		return []string{"@" + base, "D=M", "@" + strconv.Itoa(index), "D=D+A", "@R13", "M=D", "@SP", "AM=M-1", "D=M", "@R13", "A=M", "M=D"}
	}

	switch segment {
	case "constant":
		code = []string{"@SP", "AM=M-1", "D=M", "@" + strconv.Itoa(index), "M=D"}
	case "argument":
		code = pop("ARG")
	case "local":
		code = pop("LCL")
	case "this":
		code = pop("THIS")
	case "that":
		code = pop("THAT")
	case "pointer":
		code = []string{"@R3", "D=A", "@" + strconv.Itoa(index), "D=D+A", "@R13", "M=D", "@SP", "AM=M-1", "D=M", "@R13", "A=M", "M=D"}
	case "temp":
		code = []string{"@R5", "D=A", "@" + strconv.Itoa(index), "D=D+A", "@R13", "M=D", "@SP", "AM=M-1", "D=M", "@R13", "A=M", "M=D"}
	case "static":
		code = []string{"@SP", "AM=M-1", "D=M", "@" + cw.name + "." + strconv.Itoa(index), "M=D"}
	default:
		return fmt.Errorf("undefined segment: %s", segment)
	}
	cw.fPrintln(strings.Join(code, "\n"))

	return nil
}

func (cw *CodeWriter)writeArithmetic(command string) error {
	unary := func(ope string) string {
		return strings.Join([]string{"@SP", "A=M-1", ope}, "\n")
	}

	binary := func(ope string) string {
		return strings.Join([]string{"@SP", "AM=M-1", "D=M", "A=A-1", ope}, "\n")
	}

	relational := func(ope string) string {
		label := strconv.Itoa(cw.labelCnt)
		cw.labelCnt++
		code := []string{
			binary("M=M-D"), // x - y

			"D=M",
			"@TRUE" + label,
			"D;" + ope, // if (x - y operator 0) goto TRUE else goto FALSE
			"@FALSE" + label,
			"0;JMP",

			"(TRUE" + label + ")",
			"D=-1",
			"@END" + label,
			"0;JMP",

			"(FALSE" + label + ")",
			"D=0",

			"(END" + label + ")",
			"@SP",
			"A=M-1",
			"M=D",
		}
		return strings.Join(code, "\n")
	}

	switch command{
	case "add": // x + y
		cw.fPrintln(binary("M=D+M"))
	case "sub": // x - y
		cw.fPrintln(binary("M=M-D"))
	case "neg": // -y
		cw.fPrintln(unary("M=-M"))
	case "eq": // if x == y return true else return false
		cw.fPrintln(relational("JEQ"))
	case "gt": // if x > y return true else return false
		cw.fPrintln(relational("JGT"))
	case "lt": // if x < y return true else return false
		cw.fPrintln(relational("JLT"))
	case "and": // x & y
		cw.fPrintln(binary("M=D&M"))
	case "or": // x || y
		cw.fPrintln(binary("M=D|M"))
	case "not":	// !y
		cw.fPrintln(unary("M=!M"))
	}
	return nil
}
