package main

import (
	"os"
	"strconv"
	"strings"
)


func main() {
	i := 0
	p := NewParser("Prog.asm")
	for p.Scan() {
		p.Advance()
		if p.Command == "" {
			continue
		}
		switch p.CommandType {
		case L:
			p.St.AddEntry(p.Command, i)
		default:
			i++
		}
	}
	p.File.Close()
	p.reLoadFile("Prog.asm")

	out, _ := os.Create("Prog.hack")
	defer out.Close()

	for p.Scan() {
		p.Advance()
		if p.Command == "" {
			continue
		}
		switch p.CommandType {
		case A:
			intToCmd := func(v int) string {
				cmd := []string{"0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0"}
				for i := len(cmd) - 1; i >= 0; i-- {
					cmd[i] = strconv.Itoa(v & 1)
					v >>= 1
				}
				return strings.Join(cmd, "")
			}
			value, err := strconv.Atoi(p.Command)
			if err != nil {
				if v, ok := p.St.GetAddress(p.Command); ok {
					value = v
				} else {
					value = p.St.AddVariable(p.Command)
				}
			}
			out.WriteString(intToCmd(value) + "\n")
		case C:
			cmd := "111" + Comp(p.Comp) + Dest(p.Dest) + Jump(p.Jump)
			out.WriteString(cmd + "\n")
		default:
			break
		}
	}
	p.File.Close()
}
