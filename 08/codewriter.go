package main

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type CodeWriter struct{
	commands []Command
	writer io.Writer
	name string
	currentFunctionName string
}

var labelCnt = 0
var labelRegexp = regexp.MustCompile(`^[a-zA-Z_.:][a-zA-Z0-9_.:]+$`)

func NewCodeWriter(commands []Command, writer io.Writer, name string) *CodeWriter {
	return &CodeWriter{commands: commands, writer: writer, name: name, currentFunctionName: ""}
}

func (cw *CodeWriter)BootstrapCode() error {
	cw.fPrintln("@256\nD=A\n@SP\nM=D")
	if err := cw.writeCall("Sys.init", 0); err != nil {
		return err
	}
	return nil
}

func (cw *CodeWriter)GenerateCode() error {
	for _, c := range cw.commands {
		switch c.CommandType {
		case CArithmetic:
			if err := cw.writeArithmetic(c.command); err != nil {
				return err
			}
		case CPush:
			if err := cw.writePush(c.arg1, c.arg2); err != nil {
				return err
			}
		case CPop:
			if err := cw.writePop(c.arg1, c.arg2); err != nil {
				return err
			}
		case CLabel:
			if err := cw.writeLabel(c.arg1); err != nil {
				return err
			}
		case CGoto:
			if err := cw.writeGoto(c.arg1); err != nil {
				return err
			}
		case CIfGoto:
			if err := cw.writeIfGoto(c.arg1); err != nil {
				return err
			}
		case CFunction:
			cw.currentFunctionName = c.arg1
			if err := cw.writeFunction(c.arg1, c.arg2); err != nil {
				return err
			}
		case CReturn:
			if err := cw.writeReturn(); err != nil {
				return err
			}
		case CCall:
			if err := cw.writeCall(c.arg1, c.arg2); err != nil {
				return err
			}
		}
	}
	return nil
}

func (cw *CodeWriter)fPrintln(a string) {
	fmt.Fprintln(cw.writer, a)
}

func (cw *CodeWriter)push() string {
	return "@SP\nA=M\nM=D\n@SP\nM=M+1"
}

func (cw *CodeWriter)writePush(segment string, index int) error {
	var code []string

	// load index into D register
	loadIndex := func() []string {
		return []string{"@" + strconv.Itoa(index), "D=A"}
	}

	// load segment[index] into D register
	loadMemData := func(base string) []string {
		return append(loadIndex(),  []string{"@" + base, "A=D+M", "D=M"}...)
	}

	switch segment {
	case "constant":
		code = loadIndex()
	case "argument":
		code = loadMemData("ARG")
	case "local":
		code = loadMemData("LCL")
	case "this":
		code = loadMemData("THIS")
	case "that":
		code = loadMemData("THAT")
	case "pointer":
		code = append(loadIndex(), []string{"@R3", "A=D+A", "D=M"}...)
	case "temp":
		code = append(loadIndex(), []string{"@R5", "A=D+A", "D=M"}...)
	case "static":
		code = []string{"@" + cw.name + "." + strconv.Itoa(index), "D=M"}
	default:
		return fmt.Errorf("undefined segment: %s", segment)
	}
	code = append(code, cw.push())

	cw.fPrintln(strings.Join(code, "\n"))
	return nil
}

func (cw *CodeWriter)pop() string {
	return "@SP\nAM=M-1\nD=M"
}

func (cw *CodeWriter)writePop(segment string, index int) error {
	var code []string

	pop := func(base string) []string {
		return []string{"@" + base, "D=M", "@" + strconv.Itoa(index), "D=D+A", "@R13", "M=D", cw.pop(), "@R13", "A=M", "M=D"}
	}

	switch segment {
	case "constant":
		code = []string{cw.pop(), "@" + strconv.Itoa(index), "M=D"}
	case "argument":
		code = pop("ARG")
	case "local":
		code = pop("LCL")
	case "this":
		code = pop("THIS")
	case "that":
		code = pop("THAT")
	case "pointer":
		code = []string{"@R3", "D=A", "@" + strconv.Itoa(index), "D=D+A", "@R13", "M=D", cw.pop(), "@R13", "A=M", "M=D"}
	case "temp":
		code = []string{"@R5", "D=A", "@" + strconv.Itoa(index), "D=D+A", "@R13", "M=D", cw.pop(), "@R13", "A=M", "M=D"}
	case "static":
		code = []string{cw.pop(), "@" + cw.name + "." + strconv.Itoa(index), "M=D"}
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
		return strings.Join([]string{cw.pop(), "A=A-1", ope}, "\n")
	}

	relational := func(ope string) string {
		label := strconv.Itoa(labelCnt)
		labelCnt++
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

func (cw *CodeWriter)validateLabelName(name string) bool {
	return labelRegexp.MatchString(name)
}

func (cw *CodeWriter)generateLabel(dest string) string {
	return cw.currentFunctionName + "$" + dest
}

func (cw *CodeWriter)writeLabel(dest string) error {
	if ok := cw.validateLabelName(dest); !ok {
		return fmt.Errorf("invalid label name: %s", dest)
	}

	cw.fPrintln("(" + cw.generateLabel(dest) + ")")
	return nil
}

func (cw *CodeWriter)writeGoto(dest string) error {
	if ok := cw.validateLabelName(dest); !ok {
		return fmt.Errorf("invalid label name: %s", dest)
	}
	cw.fPrintln(strings.Join([]string{"@" + cw.generateLabel(dest), "0;JMP"}, "\n"))
	return nil
}

func (cw *CodeWriter)writeIfGoto(dest string) error {
	if ok := cw.validateLabelName(dest); !ok {
		return fmt.Errorf("invalid label name: %s", dest)
	}

	cw.fPrintln(strings.Join([]string{"@SP", "AM=M-1", "D=M", "@" + cw.generateLabel(dest), "D;JNE"}, "\n"))
	return nil
}

func (cw *CodeWriter)writeFunction(f string, k int) error {
	cw.fPrintln("(" + f + ")")
	for i := 0; i < k; i++ {
		cw.fPrintln(strings.Join([]string{"@0", "D=A", cw.push()}, "\n"))
	}
	return nil
}

func (cw *CodeWriter)writeReturn() error {
	code := []string{
		"@LCL", "D=M", "@R13", "M=D",  // tmp = LCL
		"@5", "D=D-A", "A=D", "D=M", "@R14", "M=D", // RET = *(tmp - 5)
		cw.pop(), "@ARG", "A=M", "M=D", // *ARG = pop()
		"@ARG", "D=M", "@SP", "M=D+1", // SP = ARG + 1
		"@R13", "AM=M-1", "D=M", "@THAT", "M=D", // THAT = *(tmp - 1)
		"@R13", "AM=M-1", "D=M", "@THIS", "M=D", // THIS = *(tmp - 2)
		"@R13", "AM=M-1", "D=M", "@ARG", "M=D", // ARG = *(tmp - 3)
		"@R13", "AM=M-1", "D=M", "@LCL", "M=D", // LCL = *(tmp - 4)
		"@R14", "A=M", "0;JMP", // goto RET
	}
	cw.fPrintln(strings.Join(code, "\n"))
	return nil
}

func (cw *CodeWriter)writeCall(f string, n int) error {
	returnAddress := "call" + strconv.Itoa(labelCnt)
	labelCnt++
	code := []string{
		"@" + returnAddress, "D=A", cw.push(),
		"@LCL", "D=M", cw.push(),
		"@ARG", "D=M", cw.push(),
		"@THIS", "D=M", cw.push(),
		"@THAT", "D=M", cw.push(),
		"@SP", "D=M", "@" + strconv.Itoa(n), "D=D-A", "@5", "D=D-A", "@ARG", "M=D",
		"@SP", "D=M", "@LCL", "M=D",
		"@" + f, "0;JMP",
		"(" + returnAddress + ")",
	}
	cw.fPrintln(strings.Join(code, "\n"))
	return nil
}
