package vmwriter

import (
	"fmt"
	"io"
)

type Segment int
type Command int

const (
	Const Segment = iota
	Arg
	Local
	Static
	This
	That
	Pointer
	Temp
)

const (
	Add Command = iota
	Sub
	Neg
	Eq
	Gt
	Lt
	And
	Or
	Not
)

var segmentMap = map[Segment]string{
	Const:   "constant",
	Arg:     "argument",
	Local:   "local",
	Static:  "static",
	This:    "this",
	That:    "that",
	Pointer: "pointer",
	Temp:    "temp",
}

var commandMap = map[Command]string{
	Add: "add",
	Sub: "sub",
	Neg: "neg",
	Eq:  "eq",
	Gt:  "gt",
	Lt:  "lt",
	And: "and",
	Or:  "or",
	Not: "not",
}

type VMWriter struct {
	out io.Writer
}

func NewVMWriter(out io.Writer) *VMWriter {
	return &VMWriter{out: out}
}

func (v *VMWriter) WritePush(segment Segment, index int) {
	fmt.Fprintf(v.out, "push %s %d\n", segmentMap[segment], index)
}

func (v *VMWriter) WritePop(segment Segment, index int) {
	fmt.Fprintf(v.out, "pop %s %d\n", segmentMap[segment], index)
}

func (v *VMWriter) WriteArithmetic(cmd Command) {
	fmt.Fprintf(v.out, "%s\n", commandMap[cmd])
}

func (v *VMWriter) WriteFunction(name string, nLocals int) {
	fmt.Fprintf(v.out, "function %s %d\n", name, nLocals)
}

func (v *VMWriter) WriteReturn() {
	fmt.Fprintf(v.out, "return\n")
}

func (v *VMWriter) WriteCall(name string, nArgs int) {
	fmt.Fprintf(v.out, "call %s %d\n", name, nArgs)
}

func (v *VMWriter) WriteIf(label string) {
	fmt.Fprintf(v.out, "if-goto %s\n", label)
}

func (v *VMWriter) WriteGoto(label string) {
	fmt.Fprintf(v.out, "goto %s\n", label)
}

func (v *VMWriter) WriteLabel(label string) {
	fmt.Fprintf(v.out, "label %s\n", label)
}
