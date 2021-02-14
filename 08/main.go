package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("missing file or directory argument")
	}

	fPath := args[1]

	fInfo, err := os.Stat(fPath)
	if err != nil {
		log.Fatal(err)
	}

	if fInfo.IsDir() {
		out, err := os.Create(path.Join(fPath, path.Base(fPath) + ".asm"))
		defer out.Close()
		if err != nil {
			log.Fatal(err)
		}

		fInfos, err := ioutil.ReadDir(fPath)
		if err != nil {
			log.Fatal(err)
		}

		locs := pickVMFileLocations(fInfos, fPath)
		for i, loc := range locs {
			file, err := os.Open(loc)
			if err != nil {
				log.Fatal(err)
			}

			p := NewParser(file)
			cmds, err := p.Parse()
			file.Close()
			if err != nil {
				log.Fatal(err)
			}

			trimmedName := strings.TrimSuffix(path.Base(file.Name()), ".vm")
			writer := NewCodeWriter(cmds, out, trimmedName)
			if i == 0 {
				if err := writer.BootstrapCode(); err != nil {
					log.Fatal(err)
				}
			}
			err = writer.GenerateCode()
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		trimmedName := strings.TrimSuffix(fPath, ".vm")
		out, err := os.Create(trimmedName + ".asm")
		defer out.Close()
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Open(fPath)
		defer file.Close()
		if err != nil {
			log.Fatal(err)
		}

		p := NewParser(file)
		cmds, err := p.Parse()
		if err != nil {
			log.Fatal(err)
		}

		writer := NewCodeWriter(cmds, out, trimmedName)
		err = writer.GenerateCode()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func pickVMFileLocations(fInfos []os.FileInfo, fPath string) (locs []string) {
	for _, f := range fInfos {
		name := f.Name()
		if strings.HasSuffix(name, ".vm") && !f.IsDir() {
			locs = append(locs, path.Join(fPath, name))
		}
	}
	return locs
}
