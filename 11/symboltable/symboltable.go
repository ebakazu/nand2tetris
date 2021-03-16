package symboltable

import (
	"github.com/ebakazu/nand2tetris/11/vmwriter"
)

type Property int

const (
	Static Property = iota
	Field
	Arg
	Var
	None
)

type table struct {
	typeName string
	property Property
	idx      int
}

type SymbolTable struct {
	class       map[string]table
	subroutine  map[string]table
	propertyCnt map[Property]int
}

func NewSymbolTable() *SymbolTable {
	cnt := map[Property]int{
		Static: 0,
		Field:  0,
		Arg:    0,
		Var:    0,
	}
	return &SymbolTable{class: map[string]table{}, subroutine: map[string]table{}, propertyCnt: cnt}
}

func (st *SymbolTable) ResetSubroutineTable() {
	st.propertyCnt[Arg] = 0
	st.propertyCnt[Var] = 0
	st.subroutine = map[string]table{}
}

func (st *SymbolTable) Define(variableName string, typeName string, kind Property) {
	idx := st.propertyCnt[kind]
	st.propertyCnt[kind]++

	if kind == Static || kind == Field {
		st.class[variableName] = table{typeName: typeName, property: kind, idx: idx}
		return
	}

	if kind == Arg || kind == Var {
		st.subroutine[variableName] = table{typeName: typeName, property: kind, idx: idx}
		return
	}
}

func (st *SymbolTable) VarCount(kind Property) int {
	return st.propertyCnt[kind]
}

func (st *SymbolTable) KindOf(name string) Property {
	if t, ok := st.subroutine[name]; ok {
		return t.property
	}

	if t, ok := st.class[name]; ok {
		return t.property
	}

	return None
}

func (st *SymbolTable) TypeOf(name string) string {
	if t, ok := st.subroutine[name]; ok {
		return t.typeName
	}

	if t, ok := st.class[name]; ok {
		return t.typeName
	}

	return ""
}

func (st *SymbolTable) IndexOf(name string) int {
	if t, ok := st.subroutine[name]; ok {
		return t.idx
	}

	if t, ok := st.class[name]; ok {
		return t.idx
	}

	return -1
}

func PropertyToSegment(prop Property) vmwriter.Segment {
	switch prop {
	case Static:
		return vmwriter.Static
	case Field:
		return vmwriter.This
	case Arg:
		return vmwriter.Arg
	case Var:
		return vmwriter.Local
	}

	return -1
}
