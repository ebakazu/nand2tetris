package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type commandType int

const (
	CArithmetic commandType = iota
	CPush
	CPop
	CLabel
	CGoto
	CIfGoto
	CFunction
	CReturn
	CCall
)

type Command struct {
	CommandType commandType
	command     string
	arg1        string
	arg2        int
}

var commandTable = map[string]commandType{
	"add":  CArithmetic,
	"sub":  CArithmetic,
	"neg":  CArithmetic,
	"eq":   CArithmetic,
	"gt":   CArithmetic,
	"lt":   CArithmetic,
	"and":  CArithmetic,
	"or":   CArithmetic,
	"not":  CArithmetic,
	"push": CPush,
	"pop":  CPop,
	"label": CLabel,
	"goto": CGoto,
	"if-goto": CIfGoto,
	"function": CFunction,
	"return": CReturn,
	"call": CCall,
}

type Parser struct {
	reader io.Reader
}

func NewParser(reader io.Reader) *Parser {
	return &Parser{reader: reader}
}

func (p *Parser)Parse() ([]Command, error) {
	var commands []Command

	s := bufio.NewScanner(p.reader)
	for s.Scan() {
		txt := s.Text()
		txt = strings.Split(txt, "//")[0]
		txt = strings.TrimSpace(txt)

		if txt == "" {
			continue
		}

		var c, arg1 string
		var arg2 int

		n, _ := fmt.Sscanf(txt, "%s %s %d", &c, &arg1, &arg2)
		if n <= 0 {
			return nil, fmt.Errorf("invalid command: %s", txt)
		}

		cType, ok := commandTable[c]
		if !ok {
			return nil, fmt.Errorf("invalid command: %s", txt)
		}

		commands = append(commands, Command{CommandType: cType, command: c, arg1: arg1, arg2: arg2})
	}
	return commands, nil
}
