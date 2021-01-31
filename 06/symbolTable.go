package main

import (
	"log"
)

type SymbolTable struct {
	table map[string]int
	ramAddress int
}

func NewSymbolTable() *SymbolTable {
	table := map[string]int{
		"SP": 0,
		"LCL": 1,
		"ARG": 2,
		"THIS": 3,
		"THAT": 4,
		"R0": 0,
		"R1": 1,
		"R2": 2,
		"R3": 3,
		"R4": 4,
		"R5": 5,
		"R6": 6,
		"R7": 7,
		"R8": 8,
		"R9": 9,
		"R10": 10,
		"R11": 11,
		"R12": 12,
		"R13": 13,
		"R14": 14,
		"R15": 15,
		"SCREEN": 16384,
		"KBD": 24576,
	}
	return &SymbolTable{table: table, ramAddress: 16}
}

func (st *SymbolTable)AddEntry(symbol string, address int) {
	if _, ok := st.table[symbol]; ok {
		log.Fatalf("%s already exists", symbol)
	}
	st.table[symbol] = address
}

func (st *SymbolTable)AddVariable(symbol string) int {
	if _, ok := st.table[symbol]; ok {
		log.Fatalf("%s already exists", symbol)
	}
	st.table[symbol] = st.ramAddress
	st.ramAddress++

	return st.ramAddress - 1
}

func (st *SymbolTable)GetAddress(symbol string) (int, bool) {
	address, ok := st.table[symbol]
	return address, ok
}
