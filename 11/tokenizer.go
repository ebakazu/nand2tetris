package main

import (
	"errors"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

type tokenType int
type keywordType int

const (
	Keyword tokenType = iota
	Symbol
	Identifier
	IntConst
	StringConst
)

var tokenTypeMap = map[tokenType]string{
	Keyword:     "keyword",
	Symbol:      "symbol",
	Identifier:  "identifier",
	IntConst:    "integerConstant",
	StringConst: "stringConstant",
}

const (
	Class keywordType = iota
	Method
	Function
	Constructor
	Int
	Boolean
	Char
	Void
	Var
	Static
	Field
	Let
	Do
	If
	Else
	While
	Return
	True
	False
	Null
	This
	Eof
)

var strToKeywordType = map[string]keywordType{
	"class":       Class,
	"method":      Method,
	"function":    Function,
	"constructor": Constructor,
	"int":         Int,
	"boolean":     Boolean,
	"char":        Char,
	"void":        Void,
	"var":         Var,
	"static":      Static,
	"field":       Field,
	"let":         Let,
	"do":          Do,
	"if":          If,
	"else":        Else,
	"while":       While,
	"return":      Return,
	"true":        True,
	"false":       False,
	"null":        Null,
	"this":        This,
}

var keywords = []string{"class", "constructor", "function", "method", "field", "static", "var", "int", "char", "boolean", "void", "true", "false", "null", "this", "let", "do", "if", "else", "while", "return"}
var symbols = []string{"{", "}", "(", ")", "[", "]", ".", ",", ";", "+", "-", "*", "/", "&", "|", "<", ">", "=", "~"}

type Tokenizer struct {
	writer io.Writer
	src    []byte
	srcIdx int
}

type Token struct {
	tokenType   tokenType
	value       string
	keywordType keywordType
}

func NewTokenizer(writer io.Writer, src []byte) *Tokenizer {
	return &Tokenizer{writer: writer, src: src, srcIdx: 0}
}

func (t *Tokenizer) get() (byte, bool) {
	if t.srcIdx < len(t.src) {
		return t.src[t.srcIdx], true
	}
	return 0, false
}

func (t *Tokenizer) Tokenize() error {
	if _, err := fmt.Fprintf(t.writer, "<tokens>\n"); err != nil {
		return err
	}

	for {
		c, ok := t.get()

		if !ok {
			break
		}

		if unicode.IsSpace(rune(c)) {
			t.srcIdx++
			continue
		}

		if c == '/' {
			t.srcIdx++
			c2, ok := t.get()
			if !ok {
				return errors.New("unexpect EOF")
			}

			if c2 == '/' {
				if err := t.findNewline(); err != nil {
					return err
				}
				t.srcIdx++
				continue
			}

			if c2 == '*' {
				if err := t.findEndOfComment(); err != nil {
					return err
				}
				t.srcIdx++
				continue
			}
		}

		if c == '"' {
			if v, ok := t.searchStrConst(); ok {
				if err := t.writeToken(StringConst, string(v)); err != nil {
					return err
				}
				t.srcIdx++
			} else {
				return errors.New(`expect '"', but find new line`)
			}
			continue
		}

		if sliceContain(string(c), symbols) {
			if err := t.writeToken(Symbol, string(c)); err != nil {
				return err
			}
			t.srcIdx++
			continue
		}

		if v, ok := t.isKeyWord(); ok {
			if err := t.writeToken(Keyword, v); err != nil {
				return err
			}
			t.srcIdx += utf8.RuneCountInString(v)
			continue
		}

		if unicode.IsDigit(rune(c)) {
			if v, ok := t.searchDigit(); ok {
				if err := t.writeToken(IntConst, string(v)); err != nil {
					return err
				}

				t.srcIdx++
			} else {
				return fmt.Errorf("invalid digit")
			}
			continue
		}

		if unicode.IsLetter(rune(c)) {
			if v, ok := t.searchIdentifier(); ok {
				if err := t.writeToken(Identifier, string(v)); err != nil {
					return err
				}

				t.srcIdx++
			} else {
				return fmt.Errorf("invalid identifier")
			}
			continue
		}
	}

	if _, err := fmt.Fprintf(t.writer, "</tokens>\n"); err != nil {
		return err
	}

	return nil
}

func (t *Tokenizer) writeToken(tt tokenType, value string) error {
	tag := tokenTypeMap[tt]

	if tt == Symbol {
		value = escapeSymbol(value)
	}

	if _, err := fmt.Fprintf(t.writer, "<%s> %s </%s>\n", tag, value, tag); err != nil {
		return err
	}

	return nil
}

func (t *Tokenizer) findNewline() error {
	for {
		if v, ok := t.get(); v == '\n' || !ok {
			return nil
		}
		t.srcIdx++
	}
}

func (t *Tokenizer) findEndOfComment() error {
	var c, newC byte
	err := errors.New("syntax error: expect '*/', but find EOF")

	c, ok := t.get()
	if !ok {
		return err
	}

	t.srcIdx++

	for {
		newC, ok = t.get()
		if !ok {
			return err
		}

		if c == '*' && newC == '/' {
			return nil
		}

		c = newC
		t.srcIdx++
	}
}

func (t *Tokenizer) isKeyWord() (string, bool) {
	for _, v := range keywords {
		if string(t.src[t.srcIdx:t.srcIdx+len(v)]) == v && !unicode.IsLetter(rune(t.src[t.srcIdx+len(v)])) {
			return v, true
		}
	}
	return "", false
}

func (t *Tokenizer) searchIdentifier() ([]byte, bool) {
	r := make([]byte, 0, 0)

	for {
		v, _ := t.get()
		if unicode.IsLetter(rune(v)) || unicode.IsDigit(rune(v)) || v == '_' {
			r = append(r, v)
		} else {
			t.srcIdx--
			break
		}
		t.srcIdx++
	}

	if len(r) > 0 {
		return r, true
	}

	return nil, false
}

func (t *Tokenizer) searchDigit() ([]byte, bool) {
	r := make([]byte, 0, 0)

	for {
		v, _ := t.get()
		if unicode.IsDigit(rune(v)) {
			r = append(r, v)
		} else {
			t.srcIdx--
			break
		}
		t.srcIdx++
	}

	if len(r) > 0 {
		return r, true
	}

	return nil, false
}

func (t *Tokenizer) searchStrConst() ([]byte, bool) {
	r := make([]byte, 0, 0)
	t.srcIdx++

	for {
		v, ok := t.get()

		if !ok || v == '\n' {
			return nil, false
		}

		if v != '\n' && v != '"' {
			r = append(r, v)
		}

		if v == '"' {
			return r, true
		}

		t.srcIdx++
	}

}

func escapeSymbol(symbol string) string {
	if symbol == "<" {
		return "&lt;"
	}

	if symbol == ">" {
		return "&gt;"
	}

	if symbol == "&" {
		return "&amp;"
	}

	return symbol
}

func decodeSymbol(symbol string) string {
	if symbol == "&lt;" {
		return "<"
	}

	if symbol == "&gt;" {
		return ">"
	}

	if symbol == "&amp;" {
		return "&"
	}

	return symbol
}

func sliceContain(w string, s []string) bool {
	for _, v := range s {
		if w == v {
			return true
		}
	}
	return false
}
