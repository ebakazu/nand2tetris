package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Parser struct {
	out       io.Writer
	tokens    []Token
	tokensIdx int
	buf       *bytes.Buffer
}

type TokenCompiler func() error

func NewParser(out io.Writer) *Parser {
	buf := bytes.NewBuffer([]byte{})
	return &Parser{out: out, tokens: []Token{}, tokensIdx: 0, buf: buf}
}

func (p *Parser) ReadTokenFile(fPath string) error {
	f, err := os.Open(fPath)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(f)

	for s.Scan() {
		txt := s.Text()
		if txt == "<tokens>" || txt == "</tokens>" {
			continue
		}

		a := strings.Split(txt, "<")
		b := strings.Split(a[1], ">")
		key1, value := b[0], strings.TrimSpace(b[1])

		switch key1 {
		case "tokens":
			continue
		case "keyword":
			keywordType, ok := strToKeywordType[value]
			if !ok {
				return fmt.Errorf("invalid token: %s", txt)
			}
			p.tokens = append(p.tokens, Token{tokenType: Keyword, value: value, keywordType: keywordType})

		case "symbol":
			p.tokens = append(p.tokens, Token{tokenType: Symbol, value: decodeSymbol(value)})

		case "identifier":
			p.tokens = append(p.tokens, Token{tokenType: Identifier, value: value})

		case "integerConstant":
			p.tokens = append(p.tokens, Token{tokenType: IntConst, value: value})

		case "stringConstant":
			p.tokens = append(p.tokens, Token{tokenType: StringConst, value: value})
		default:
			return fmt.Errorf("invalid key: %s, text: %s", key1, txt)
		}

	}
	return nil
}

func (p *Parser) get() (*Token, error) {
	if p.tokensIdx == len(p.tokens) {
		return nil, nil
	}
	if p.tokensIdx > len(p.tokens) {
		return nil, errors.New("parser.tokens: index out of range")
	}
	return &p.tokens[p.tokensIdx], nil
}

func (p *Parser) next() {
	p.tokensIdx++
}

func (p *Parser) back() {
	p.tokensIdx--
}

func (p *Parser) expect(expect Token) (*Token, error) {
	actual, err := p.get()
	if err != nil {
		return nil, err
	}

	if expect.tokenType == Keyword && actual.keywordType != expect.keywordType {
		return nil, fmt.Errorf("expect keywordType: %d, but %d", expect.keywordType, actual.keywordType)
	}

	if expect.value != "" && actual.value != expect.value {
		return nil, fmt.Errorf("expcet value: %s, but %s", expect.value, actual.value)
	}

	if actual.tokenType != expect.tokenType {
		return nil, fmt.Errorf("expect tokenType: %s, but %s", tokenTypeMap[expect.tokenType], tokenTypeMap[actual.tokenType])
	}

	return actual, nil
}

func (p *Parser) compileSymbol(s string) error {
	token, err := p.expect(Token{tokenType: Symbol, value: s})
	if err != nil {
		return err
	}

	if err := p.writeToken(Symbol, token.value); err != nil {
		return err
	}

	return nil
}

func (p *Parser) compileIdentifier() error {
	token, err := p.expect(Token{tokenType: Identifier})
	if err != nil {
		return err
	}

	if err := p.writeToken(Identifier, token.value); err != nil {
		return err
	}

	return nil
}

func (p *Parser) compileIntConst() error {
	token, err := p.expect(Token{tokenType: IntConst})
	if err != nil {
		return err
	}
	if err := p.writeToken(IntConst, token.value); err != nil {
		return err
	}

	return nil
}

func (p *Parser) compileStringConst() error {
	token, err := p.expect(Token{tokenType: StringConst})
	if err != nil {
		return err
	}

	if err := p.writeToken(StringConst, token.value); err != nil {
		return err
	}

	return nil
}

func (p *Parser) compileKeyword(keywordType keywordType) error {
	token, err := p.expect(Token{tokenType: Keyword, keywordType: keywordType})
	if err != nil {
		return err
	}

	if err := p.writeToken(Keyword, token.value); err != nil {
		return err
	}
	return nil
}

func (p *Parser) writeToken(tt tokenType, value string) error {
	tag := tokenTypeMap[tt]

	if tt == Symbol {
		value = escapeSymbol(value)
	}

	if _, err := fmt.Fprintf(p.out, "<%s> %s </%s>\n", tag, value, tag); err != nil {
		return err
	}

	return nil
}

func (p *Parser) writeTag(value string, end bool) error {
	var format string
	if end {
		format = "</%s>\n"
	} else {
		format = "<%s>\n"
	}

	if _, err := fmt.Fprintf(p.out, format, value); err != nil {
		return err
	}

	return nil
}

func (p *Parser) Parse() error {
	for {
		t, err := p.get()
		if err != nil {
			return err
		}

		if t == nil {
			return nil
		}

		if err := p.class(); err != nil {
			return err
		}
		p.next()
	}
}

func (p *Parser) class() error {
	if err := p.writeTag("class", false); err != nil {
		return err
	}

	if err := p.compileKeyword(Class); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("{"); err != nil {
		return err
	}

	classVarDec := []string{"static", "field"}
	p.next()
	for {
		t, err := p.get()
		if err != nil {
			return err
		}

		if ok := sliceContain(t.value, classVarDec); !ok {
			break
		}

		if err := p.classVarDec(); err != nil {
			return err
		}

		p.next()
	}

	subroutineDec := []string{"constructor", "function", "method"}
	for {
		t, err := p.get()
		if err != nil {
			return err
		}

		if ok := sliceContain(t.value, subroutineDec); !ok {
			break
		}

		if err := p.subroutine(); err != nil {
			return nil
		}

		p.next()
	}

	if err := p.compileSymbol("}"); err != nil {
		return err
	}

	if err := p.writeTag("class", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) keywordTypeContain(keywordTypes []keywordType) (*Token, bool) {
	for _, keywordType := range keywordTypes {
		token, err := p.expect(Token{tokenType: Keyword, keywordType: keywordType})
		if err == nil {
			return token, true
		}
	}
	return nil, false
}

func (p *Parser) classVarDec() error {
	if err := p.writeTag("classVarDec", false); err != nil {
		return err
	}

	keywordTypes := []keywordType{Static, Field}
	token, ok := p.keywordTypeContain(keywordTypes)
	if !ok {
		return fmt.Errorf("expect %T, but not found", keywordTypes)
	}

	if err := p.compileKeyword(token.keywordType); err != nil {
		return err
	}

	p.next()
	if err := p.typeName(); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	for {
		p.next()
		t, err := p.get()
		if err != nil {
			return err
		}

		if t.value != "," {
			break
		}

		if err := p.compileSymbol(","); err != nil {
			return err
		}

		p.next()
		if err := p.compileIdentifier(); err != nil {
			return err
		}
	}

	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	if err := p.writeTag("classVarDec", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) typeName() error {
	if token, ok := p.keywordTypeContain([]keywordType{Int, Char, Boolean}); ok {
		if err := p.writeToken(Keyword, token.value); err != nil {
			return err
		}
		return nil
	}

	if err := p.compileIdentifier(); err != nil {
		return err
	}

	return nil
}

func (p *Parser) subroutine() error {
	if err := p.writeTag("subroutineDec", false); err != nil {
		return err
	}

	keywordTypes := []keywordType{Constructor, Function, Method}
	token, ok := p.keywordTypeContain(keywordTypes)
	if ok {
		if err := p.compileKeyword(token.keywordType); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("expect %T, but not found", keywordTypes)
	}

	p.next()
	if token, ok := p.keywordTypeContain([]keywordType{Void}); ok {
		if err := p.writeToken(Keyword, token.value); err != nil {
			return err
		}
	} else if err := p.typeName(); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("("); err != nil {
		return err
	}

	p.next()
	ok, err := p.parameterList()
	if err != nil {
		return err
	}

	if ok {
		p.next()
	}

	if err := p.compileSymbol(")"); err != nil {
		return err
	}

	if err := p.writeTag("subroutineBody", false); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("{"); err != nil {
		return err
	}

	for {
		p.next()
		t, err := p.get()
		if err != nil {
			return err
		}

		if t.value != "var" {
			break
		}

		if err := p.varDec(); err != nil {
			return err
		}
	}

	if err := p.statements(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("}"); err != nil {
		return err
	}

	if err := p.writeTag("subroutineBody", true); err != nil {
		return err
	}

	if err := p.writeTag("subroutineDec", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parameterList() (bool, error) {
	if err := p.writeTag("parameterList", false); err != nil {
		return false, err
	}

	t, err := p.get()
	if err != nil {
		return false, err
	}

	if t.tokenType == Identifier || sliceContain(t.value, []string{"int", "char", "boolean"}) {
		if err := p.typeName(); err != nil {
			return false, err
		}
		p.next()
		if err := p.compileIdentifier(); err != nil {
			return false, err
		}

		for {
			p.next()
			t, err := p.get()
			if err != nil {
				return false, err
			}

			if t.value != "," {
				p.back()
				break
			}

			if err := p.compileSymbol(","); err != nil {
				return false, err
			}

			p.next()
			if err := p.typeName(); err != nil {
				return false, err
			}

			p.next()
			if err := p.compileIdentifier(); err != nil {
				return false, err
			}
		}

		if err := p.writeTag("parameterList", true); err != nil {
			return false, err
		}

		return true, nil
	}

	if err := p.writeTag("parameterList", true); err != nil {
		return false, err
	}

	return false, nil
}

func (p *Parser) varDec() error {
	if err := p.writeTag("varDec", false); err != nil {
		return err
	}

	if err := p.compileKeyword(Var); err != nil {
		return err
	}

	p.next()
	if err := p.typeName(); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	p.next()
	for {
		t, err := p.get()
		if err != nil {
			return err
		}

		if t.value != "," {
			break
		}

		if err := p.compileSymbol(","); err != nil {
			return err
		}

		p.next()
		if err := p.compileIdentifier(); err != nil {
			return err
		}

		p.next()
	}

	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	if err := p.writeTag("varDec", true); err != nil {
		return err
	}
	return nil
}

func (p *Parser) statements() error {
	if err := p.writeTag("statements", false); err != nil {
		return err
	}

	for {
		token, err := p.get()
		if err != nil {
			return err
		}

		statements := []string{"let", "if", "while", "do", "return"}
		if ok := sliceContain(token.value, statements); !ok {
			p.back()
			break
		}

		switch token.keywordType {
		case Let:
			if err := p.letStatement(); err != nil {
				return err
			}
		case If:
			if err := p.ifStatement(); err != nil {
				return err
			}
		case While:
			if err := p.whileStatement(); err != nil {
				return err
			}
		case Do:
			if err := p.doStatement(); err != nil {
				return err
			}
		case Return:
			if err := p.returnStatement(); err != nil {
				return err
			}
		}

		p.next()
	}

	if err := p.writeTag("statements", true); err != nil {
		return err
	}
	return nil
}

func (p *Parser) letStatement() error {
	if err := p.writeTag("letStatement", false); err != nil {
		return err
	}

	if err := p.compileKeyword(Let); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	p.next()
	t, err := p.get()
	if err != nil {
		return err
	}

	if t.value == "[" {
		if err := p.compileSymbol("["); err != nil {
			return err
		}

		p.next()
		if err := p.expression(); err != nil {
			return err
		}

		p.next()
		if err := p.compileSymbol("]"); err != nil {
			return err
		}

		p.next()
	}

	if err := p.compileSymbol("="); err != nil {
		return err
	}

	p.next()
	if err := p.expression(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	if err := p.writeTag("letStatement", true); err != nil {
		return err
	}
	return nil
}

func (p *Parser) ifStatement() error {
	if err := p.writeTag("ifStatement", false); err != nil {
		return err
	}

	if err := p.compileKeyword(If); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("("); err != nil {
		return err
	}

	p.next()
	if err := p.expression(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol(")"); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("{"); err != nil {
		return err
	}

	p.next()
	if err := p.statements(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("}"); err != nil {
		return err
	}

	p.next()
	t, err := p.get()
	if err != nil {
		return err
	}

	if t.value == "else" {
		if err := p.compileKeyword(Else); err != nil {
			return err
		}

		p.next()
		if err := p.compileSymbol("{"); err != nil {
			return err
		}

		p.next()
		if err := p.statements(); err != nil {
			return err
		}

		p.next()
		if err := p.compileSymbol("}"); err != nil {
			return err
		}
	} else {
		p.back()
	}

	if err := p.writeTag("ifStatement", true); err != nil {
		return err
	}
	return err
}

func (p *Parser) whileStatement() error {
	if err := p.writeTag("whileStatement", false); err != nil {
		return err
	}

	if err := p.compileKeyword(While); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("("); err != nil {
		return err
	}

	p.next()
	if err := p.expression(); err != nil {
		return nil
	}

	p.next()
	if err := p.compileSymbol(")"); err != nil {
		return nil
	}

	p.next()
	if err := p.compileSymbol("{"); err != nil {
		return err
	}

	p.next()
	if err := p.statements(); err != nil {
		return nil
	}

	p.next()
	if err := p.compileSymbol("}"); err != nil {
		return err
	}

	if err := p.writeTag("whileStatement", true); err != nil {
		return err
	}
	return nil
}

func (p *Parser) doStatement() error {
	if err := p.writeTag("doStatement", false); err != nil {
		return err
	}

	if err := p.compileKeyword(Do); err != nil {
		return err
	}

	p.next()
	if err := p.subroutineCall(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	if err := p.writeTag("doStatement", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) returnStatement() error {
	if err := p.writeTag("returnStatement", false); err != nil {
		return err
	}

	if err := p.compileKeyword(Return); err != nil {
		return err
	}

	p.next()
	t, err := p.get()
	if err != nil {
		return err
	}

	s := []string{"true", "false", "null", "this", "(", "-", "~"}
	if t.tokenType == IntConst || t.tokenType == StringConst || t.tokenType == Identifier || sliceContain(t.value, s) {
		if err := p.expression(); err != nil {
			return err
		}
		p.next()
	}

	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	if err := p.writeTag("returnStatement", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) expression() error {
	if err := p.writeTag("expression", false); err != nil {
		return err
	}

	if err := p.term(); err != nil {
		return err
	}

	ops := []string{"+", "-", "*", "/", "&", "|", ">", "<", "="}
	for {
		p.next()
		t, err := p.get()
		if err != nil {
			return err
		}

		if ok := sliceContain(t.value, ops); !ok {
			p.back()
			break
		}

		if err := p.op(); err != nil {
			return err
		}

		p.next()
		if err := p.term(); err != nil {
			return err
		}
	}

	if err := p.writeTag("expression", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) op() error {
	token, err := p.get()
	if err != nil {
		return err
	}

	ops := []string{"+", "-", "*", "/", "&", "|", ">", "<", "="}
	for _, op := range ops {
		if err := p.compileSymbol(op); err == nil {
			return nil
		}
	}

	return fmt.Errorf("expect operator, but, %s", token.value)
}

func (p *Parser) term() error {
	if err := p.writeTag("term", false); err != nil {
		return err
	}

	token, err := p.get()
	if err != nil {
		return err
	}

	p.next()

	nextToken, err := p.get()
	if err != nil {
		return err
	}
	p.back()

	if token.tokenType == Identifier && nextToken.value == "[" {
		if err := p.compileIdentifier(); err != nil {
			return err
		}

		p.next()
		if err := p.compileSymbol("["); err != nil {
			return err
		}

		p.next()
		if err := p.expression(); err != nil {
			return err
		}

		p.next()
		if err := p.compileSymbol("]"); err != nil {
			return err
		}
	} else if token.tokenType == Identifier && (nextToken.value == "(" || nextToken.value == ".") {
		if err := p.subroutineCall(); err != nil {
			return err
		}
	} else if token.tokenType == Identifier && err == nil {
		if err := p.compileIdentifier(); err != nil {
			return err
		}
	} else if token.value == "(" {
		if err := p.compileSymbol("("); err != nil {
			return err
		}

		p.next()
		if err := p.expression(); err != nil {
			return err
		}

		p.next()
		if err := p.compileSymbol(")"); err != nil {
			return err
		}
	} else if token.tokenType == IntConst {
		if err := p.compileIntConst(); err != nil {
			return err
		}
	} else if token.tokenType == StringConst {
		if err := p.compileStringConst(); err != nil {
			return err
		}
	} else if token.keywordType == True || token.keywordType == False || token.keywordType == Null || token.keywordType == This {
		if err := p.compileKeyword(token.keywordType); err != nil {
			return err
		}
	} else if token.tokenType == Symbol && (token.value == "-" || token.value == "~") {
		if err := p.compileSymbol(token.value); err != nil {
			return err
		}

		p.next()
		if err := p.term(); err != nil {
			return err
		}
	}

	if err := p.writeTag("term", true); err != nil {
		return err
	}

	return nil
}

func (p *Parser) expressionList() (bool, error) {
	if err := p.writeTag("expressionList", false); err != nil {
		return false, err
	}

	t, err := p.get()
	if err != nil {
		return false, err
	}

	s := []string{"true", "false", "null", "this", "(", "-", "~"}

	if t.tokenType == IntConst || t.tokenType == StringConst || t.tokenType == Identifier || sliceContain(t.value, s) {
		if err := p.expression(); err != nil {
			return false, err
		}

		p.next()
		for {
			t, err := p.get()
			if err != nil {
				return false, err
			}

			if t.value != "," {
				p.back()
				break
			}

			if err := p.compileSymbol(","); err != nil {
				return false, err
			}

			p.next()
			if err := p.expression(); err != nil {
				return false, err
			}

			p.next()
		}

		if err := p.writeTag("expressionList", true); err != nil {
			return false, err
		}

		return true, err
	}

	if err := p.writeTag("expressionList", true); err != nil {
		return false, err
	}

	return false, err
}

func (p *Parser) subroutineCall() error {
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	p.next()
	token, err := p.get()
	if err != nil {
		return err
	}

	if token.value == "." {
		if err := p.compileSymbol("."); err != nil {
			return err
		}

		p.next()
		if err := p.compileIdentifier(); err != nil {
			return err
		}

		p.next()
	}

	if err := p.compileSymbol("("); err != nil {
		return err
	}

	p.next()
	ok, err := p.expressionList()
	if err != nil {
		return err
	}

	if ok {
		p.next()
	}

	if err := p.compileSymbol(")"); err != nil {
		return err
	}

	return nil
}
