package main

import "strings"

var compMap = map[string]string{
	"0": "0101010",
	"1": "0111111",
	"-1": "0111010",
	"D": "0001100",
	"A": "0110000",
	"!D": "0001101",
	"!A": "0110001",
	"-D": "0001111",
	"-A": "0110011",
	"D+1": "0011111",
	"A+1": "0110111",
	"D-1": "0001110",
	"A-1": "0110010",
	"D+A": "0000010",
	"D-A": "0010011",
	"A-D": "0000111",
	"D&A": "0000000",
	"D|A": "0010101",
	"M": "1110000",
	"!M": "1110001",
	"-M": "1110011",
	"M+1": "1110111",
	"M-1": "1110010",
	"D+M": "1000010",
	"D-M": "1010011",
	"M-D": "1000111",
	"D&M": "1000000",
	"D|M": "1010101",
}

var jumpMap = map[string]string{
	"": "000",
	"JGT": "001",
	"JEQ": "010",
	"JGE": "011",
	"JLT": "100",
	"JNE": "101",
	"JLE": "110",
	"JMP": "111",
}

func Dest(mnemonic string) string {
	r := []string{"0", "0", "0"}
	for _, c := range mnemonic {
		switch c {
		case 'A':
			r[0] = "1"
		case 'D':
			r[1] = "1"
		case 'M':
			r[2] = "1"
		default:
			r = []string{"0", "0", "0"}
		}
	}
	return strings.Join(r, "")
}

func Comp(mnemonic string) string {
	return compMap[mnemonic]
}

func Jump(mnemonic string) string {
	return jumpMap[mnemonic]
}
