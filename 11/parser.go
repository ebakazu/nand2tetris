package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ebakazu/nand2tetris/11/symboltable"
	"github.com/ebakazu/nand2tetris/11/vmwriter"
)

type Parser struct {
	out         io.Writer
	tokens      []Token
	tokensIdx   int
	labelCnt    int
	className   string
	buf         *bytes.Buffer
	symbolTable *symboltable.SymbolTable
	vmwriter    *vmwriter.VMWriter
}

type TokenCompiler func() error

func NewParser(out io.Writer) *Parser {
	buf := bytes.NewBuffer([]byte{})
	vmWriter := vmwriter.NewVMWriter(out)
	symbolTable := symboltable.NewSymbolTable()
	return &Parser{out: out, tokens: []Token{}, tokensIdx: 0, labelCnt: 0, buf: buf, symbolTable: symbolTable, vmwriter: vmWriter}
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
		return nil, fmt.Errorf("expect value: %s, but %s", expect.value, actual.value)
	}

	if actual.tokenType != expect.tokenType {
		return nil, fmt.Errorf("expect tokenType: %s, but %s", tokenTypeMap[expect.tokenType], tokenTypeMap[actual.tokenType])
	}

	return actual, nil
}

func (p *Parser) compileSymbol(s string) error {
	_, err := p.expect(Token{tokenType: Symbol, value: s})
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) compileIdentifier() error {
	_, err := p.expect(Token{tokenType: Identifier})
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) compileIntConst() error {
	token, err := p.expect(Token{tokenType: IntConst})
	if err != nil {
		return err
	}

	intValue, _ := strconv.Atoi(token.value)
	p.vmwriter.WritePush(vmwriter.Const, intValue)

	return nil
}

func (p *Parser) compileStringConst() error {
	t, err := p.expect(Token{tokenType: StringConst})
	if err != nil {
		return err
	}

	p.vmwriter.WritePush(vmwriter.Const, len(t.value))
	p.vmwriter.WriteCall("String.new", 1)

	for _, v := range t.value {
		p.vmwriter.WritePush(vmwriter.Const, int(v))
		p.vmwriter.WriteCall("String.appendChar", 2)
	}

	return nil
}

func (p *Parser) compileKeyword(keywordType keywordType) error {
	_, err := p.expect(Token{tokenType: Keyword, keywordType: keywordType})
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) WriteArithmetic(op string) error {
	switch op {
	case "+":
		p.vmwriter.WriteArithmetic(vmwriter.Add)
	case "-":
		p.vmwriter.WriteArithmetic(vmwriter.Sub)
	case "*":
		p.vmwriter.WriteCall("Math.multiply", 2)
	case "/":
		p.vmwriter.WriteCall("Math.divide", 2)
	case "&":
		p.vmwriter.WriteArithmetic(vmwriter.And)
	case "|":
		p.vmwriter.WriteArithmetic(vmwriter.Or)
	case ">":
		p.vmwriter.WriteArithmetic(vmwriter.Gt)
	case "<":
		p.vmwriter.WriteArithmetic(vmwriter.Lt)
	case "~":
		p.vmwriter.WriteArithmetic(vmwriter.Not)
	case "=":
		p.vmwriter.WriteArithmetic(vmwriter.Eq)
	default:
		return fmt.Errorf("enexpected arithmetic: %s", op)
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
	p.symbolTable = symboltable.NewSymbolTable()

	if err := p.compileKeyword(Class); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}
	t, _ := p.get()
	className := t.value
	p.className = t.value

	p.next()
	if err := p.compileSymbol("{"); err != nil {
		return err
	}

	p.next()

	for {
		t, _ := p.get()
		if t.value != "static" && t.value != "field" {
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

		if err := p.subroutine(className); err != nil {
			return err
		}

		p.next()
	}

	if err := p.compileSymbol("}"); err != nil {
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
	keywordTypes := []keywordType{Static, Field}
	token, ok := p.keywordTypeContain(keywordTypes)
	if !ok {
		return fmt.Errorf("expect %T, but not found", keywordTypes)
	}

	var property symboltable.Property
	switch token.value {
	case "field":
		property = symboltable.Field
	case "static":
		property = symboltable.Static
	}

	p.next()
	typeName, err := p.typeName()
	if err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	t, _ := p.get()
	p.symbolTable.Define(t.value, typeName, property)

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

		t, _ = p.get()
		p.symbolTable.Define(t.value, typeName, property)
	}

	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	return nil
}

func (p *Parser) typeName() (string, error) {
	if token, ok := p.keywordTypeContain([]keywordType{Int, Char, Boolean}); ok {
		return token.value, nil
	}

	if err := p.compileIdentifier(); err != nil {
		return "", err
	}

	t, _ := p.get()
	return t.value, nil
}

func (p *Parser) subroutine(className string) error {
	p.symbolTable.ResetSubroutineTable()

	keywordTypes := []keywordType{Constructor, Function, Method}
	t, ok := p.keywordTypeContain(keywordTypes)
	if !ok {
		return fmt.Errorf("expect %T, but not found", keywordTypes)
	}

	subroutineType := t.keywordType

	p.next()

	if _, ok := p.keywordTypeContain([]keywordType{Void}); ok {
	} else if _, err := p.typeName(); err != nil {
		return err
	}

	p.next()
	subroutineName, err := p.expect(Token{tokenType: Identifier})
	if err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("("); err != nil {
		return err
	}

	if subroutineType == Method {
		p.symbolTable.Define("this", className, symboltable.Arg)
	}

	p.next()
	ok, err = p.parameterList()
	if err != nil {
		return err
	}

	if ok {
		p.next()
	}

	if err := p.compileSymbol(")"); err != nil {
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

	p.vmwriter.WriteFunction(className+"."+subroutineName.value, p.symbolTable.VarCount(symboltable.Var))
	if subroutineType == Constructor {
		p.vmwriter.WritePush(vmwriter.Const, p.symbolTable.VarCount(symboltable.Field))
		p.vmwriter.WriteCall("Memory.alloc", 1)
		p.vmwriter.WritePop(vmwriter.Pointer, 0)
	}

	if subroutineType == Method {
		p.vmwriter.WritePush(vmwriter.Arg, 0)
		p.vmwriter.WritePop(vmwriter.Pointer, 0)
	}

	if err := p.statements(); err != nil {
		return err
	}

	p.next()
	if err := p.compileSymbol("}"); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parameterList() (bool, error) {
	t, err := p.get()
	if err != nil {
		return false, err
	}

	if t.tokenType == Identifier || sliceContain(t.value, []string{"int", "char", "boolean"}) {
		typeName, err := p.typeName()
		if err != nil {
			return false, err
		}

		p.next()
		if err := p.compileIdentifier(); err != nil {
			return false, err
		}

		t, _ := p.get()
		p.symbolTable.Define(t.value, typeName, symboltable.Arg)

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
			typeName, err := p.typeName()
			if err != nil {
				return false, err
			}

			p.next()
			if err := p.compileIdentifier(); err != nil {
				return false, err
			}

			t, _ = p.get()
			p.symbolTable.Define(t.value, typeName, symboltable.Arg)
		}

		return true, nil
	}

	return false, nil
}

func (p *Parser) varDec() error {
	if err := p.compileKeyword(Var); err != nil {
		return err
	}

	p.next()
	typeName, err := p.typeName()
	if err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	t, _ := p.get()
	p.symbolTable.Define(t.value, typeName, symboltable.Var)

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

		t, _ = p.get()
		p.symbolTable.Define(t.value, typeName, symboltable.Var)

		p.next()
	}

	if err := p.compileSymbol(";"); err != nil {
		return err
	}

	return nil
}

func (p *Parser) statements() error {
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

	return nil
}

func (p *Parser) letStatement() error {
	if err := p.compileKeyword(Let); err != nil {
		return err
	}

	p.next()
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	t, _ := p.get()
	variableName := t.value
	prop := p.symbolTable.KindOf(variableName)
	idx := p.symbolTable.IndexOf(variableName)

	p.next()
	t, err := p.get()
	if err != nil {
		return err
	}

	isArray := false
	if t.value == "[" {
		isArray = true
		p.vmwriter.WritePush(symboltable.PropertyToSegment(prop), idx)

		if err := p.compileSymbol("["); err != nil {
			return err
		}

		p.next()
		if err := p.expression(); err != nil {
			return err
		}

		p.vmwriter.WriteArithmetic(vmwriter.Add)

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

	if isArray {
		p.vmwriter.WritePop(vmwriter.Temp, 0)
		p.vmwriter.WritePop(vmwriter.Pointer, 1)
		p.vmwriter.WritePush(vmwriter.Temp, 0)
		p.vmwriter.WritePop(vmwriter.That, 0)
	} else {
		p.vmwriter.WritePop(symboltable.PropertyToSegment(prop), idx)
	}

	return nil
}

func (p *Parser) ifStatement() error {
	label1 := "ELSE" + strconv.Itoa(p.labelCnt)
	label2 := "ENDIF" + strconv.Itoa(p.labelCnt)
	p.labelCnt++

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

	p.vmwriter.WriteArithmetic(vmwriter.Not)
	p.vmwriter.WriteIf(label1)

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

	p.vmwriter.WriteGoto(label2)
	p.vmwriter.WriteLabel(label1)

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

	p.vmwriter.WriteLabel(label2)

	return nil
}

func (p *Parser) whileStatement() error {
	label1 := "LOOP" + strconv.Itoa(p.labelCnt)
	label2 := "ENDLOOP" + strconv.Itoa(p.labelCnt)
	p.labelCnt++

	if err := p.compileKeyword(While); err != nil {
		return err
	}

	p.vmwriter.WriteLabel(label1)

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

	p.vmwriter.WriteArithmetic(vmwriter.Not)
	p.vmwriter.WriteIf(label2)

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

	p.vmwriter.WriteGoto(label1)
	p.vmwriter.WriteLabel(label2)

	return nil
}

func (p *Parser) doStatement() error {
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

	return nil
}

func (p *Parser) returnStatement() error {
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

	p.vmwriter.WriteReturn()

	return nil
}

func (p *Parser) expression() error {
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

		op, err := p.op()
		if err != nil {
			return err
		}

		p.next()
		if err := p.term(); err != nil {
			return err
		}

		p.WriteArithmetic(op)
	}

	return nil
}

func (p *Parser) op() (string, error) {
	token, err := p.get()
	if err != nil {
		return "", err
	}

	ops := []string{"+", "-", "*", "/", "&", "|", ">", "<", "="}
	for _, op := range ops {
		if err := p.compileSymbol(op); err == nil {
			return op, nil
		}
	}

	return "", fmt.Errorf("expect operator, but, %s", token.value)
}

func (p *Parser) term() error {
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

		t, _ := p.get()
		kind := p.symbolTable.KindOf(t.value)
		idx := p.symbolTable.IndexOf(t.value)
		p.vmwriter.WritePush(symboltable.PropertyToSegment(kind), idx)

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

		p.vmwriter.WriteArithmetic(vmwriter.Add)
		p.vmwriter.WritePop(vmwriter.Pointer, 1)
		p.vmwriter.WritePush(vmwriter.That, 0)
	} else if token.tokenType == Identifier && (nextToken.value == "(" || nextToken.value == ".") {
		if err := p.subroutineCall(); err != nil {
			return err
		}
	} else if token.tokenType == Identifier && err == nil {
		if err := p.compileIdentifier(); err != nil {
			return err
		}

		t, _ := p.get()
		prop := p.symbolTable.KindOf(t.value)
		idx := p.symbolTable.IndexOf(t.value)

		p.vmwriter.WritePush(symboltable.PropertyToSegment(prop), idx)
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
	} else if token.keywordType == True {
		p.vmwriter.WritePush(vmwriter.Const, 1)
		p.vmwriter.WriteArithmetic(vmwriter.Neg)

	} else if token.keywordType == False || token.keywordType == Null {
		p.vmwriter.WritePush(vmwriter.Const, 0)

	} else if token.keywordType == This {
		p.vmwriter.WritePush(vmwriter.Pointer, 0)
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

		switch token.value {
		case "-":
			p.vmwriter.WriteArithmetic(vmwriter.Neg)
		case "~":
			p.vmwriter.WriteArithmetic(vmwriter.Not)
		}
	}

	return nil
}

func (p *Parser) expressionList() (int, error) {
	t, err := p.get()
	if err != nil {
		return 0, err
	}

	s := []string{"true", "false", "null", "this", "(", "-", "~"}
	expressionCnt := 0

	if t.tokenType == IntConst || t.tokenType == StringConst || t.tokenType == Identifier || sliceContain(t.value, s) {
		if err := p.expression(); err != nil {
			return 0, err
		}
		expressionCnt++

		p.next()
		for {
			t, err := p.get()
			if err != nil {
				return 0, err
			}

			if t.value != "," {
				p.back()
				break
			}

			if err := p.compileSymbol(","); err != nil {
				return 0, err
			}

			p.next()
			if err := p.expression(); err != nil {
				return 0, err
			}

			expressionCnt++
			p.next()
		}

		return expressionCnt, err
	}

	return 0, err
}

func (p *Parser) subroutineCall() error {
	if err := p.compileIdentifier(); err != nil {
		return err
	}

	token, _ := p.get()
	subroutineName := token.value
	p.next()

	vKind := p.symbolTable.KindOf(token.value)
	expressionCnt := 0

	if vKind != symboltable.None {
		// instanceName.methodName
		subroutineName = p.symbolTable.TypeOf(token.value)
		vIdx := p.symbolTable.IndexOf(token.value)

		p.vmwriter.WritePush(symboltable.PropertyToSegment(vKind), vIdx)

		token, err := p.get()
		if err != nil {
			return err
		}

		if token.value != "." {
			return fmt.Errorf("expect '.', but %s", token.value)
		}

		p.next()

		token, err = p.get()
		if err != nil {
			return err
		}

		subroutineName += "." + token.value
		expressionCnt++
	} else {
		previous := token
		token, err := p.get()
		if err != nil {
			return err
		}

		switch token.value {
		case ".":
			// className.subroutineName
			p.next()
			token, err = p.get()
			if err != nil {
				return err
			}

			subroutineName += "." + token.value
		default:
			// methodName
			p.back()
			p.vmwriter.WritePush(vmwriter.Pointer, 0)
			subroutineName = p.className + "." + previous.value
			expressionCnt++
		}

	}

	p.next()
	if err := p.compileSymbol("("); err != nil {
		return err
	}

	p.next()
	cnt, err := p.expressionList()
	if err != nil {
		return err
	}

	if cnt > 0 {
		p.next()
	}

	expressionCnt += cnt

	if err := p.compileSymbol(")"); err != nil {
		return err
	}

	p.vmwriter.WriteCall(subroutineName, expressionCnt)

	return nil
}
