package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type commandType int

const (
	A commandType = iota
	C
	L
)

type Parser struct {
	File *os.File
	scanner *bufio.Scanner
	St *SymbolTable
	Command string
	CommandType commandType
	Comp string
	Dest string
	Jump string
}

func NewParser(name string) *Parser {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}

	s := bufio.NewScanner(f)
	st := NewSymbolTable()

	return &Parser{File: f, scanner: s, St: st}
}

func (p *Parser)reLoadFile(name string) {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}

	p.File = f
	p.scanner = bufio.NewScanner(f)
}

func (p *Parser) Scan() bool {
	return p.scanner.Scan()
}

func (p *Parser) Advance() {
	txt := p.scanner.Text()
	txt = strings.Split(txt, "//")[0] // trim comment
	txt = strings.TrimSpace(txt)

	if txt == "" {
		p.Command = ""
		return
	}

	if strings.HasPrefix(txt, "@") {
		p.Command = strings.Trim(txt, "@")
		p.CommandType = A
		return
	}

	if strings.HasPrefix(txt, "(") && strings.HasSuffix(txt, ")") {
		p.Command = strings.Trim(txt, "()")
		p.CommandType = L
		return
	}

	buf := ""
	isJump := false
	p.Dest = ""
	p.Comp = ""
	p.Jump = ""

	for _, c := range txt {
		if c == '=' {
			p.Dest = buf
			buf = ""
		} else if c == ';' {
			isJump = true
			p.Comp = buf
			buf = ""
		} else {
			buf += string(c)
		}
	}

	if isJump {
		p.Jump = buf
	} else {
		p.Comp = buf
	}
	p.CommandType = C
}

func (p *Parser) Symbol() string {
	switch p.CommandType {
	case A:
		return p.Command
	case L:
		return p.Command
	default:
		return ""
	}
}
