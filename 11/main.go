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
		fInfos, err := ioutil.ReadDir(fPath)
		if err != nil {
			log.Fatal(err)
		}

		locs := pickJackFileLocations(fInfos, fPath)
		for _, loc := range locs {
			if err := generate(loc); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		if err := generate(fPath); err != nil {
			log.Fatal(err)
		}
	}
}

func pickJackFileLocations(fInfos []os.FileInfo, fPath string) (locs []string) {
	for _, f := range fInfos {
		name := f.Name()
		if strings.HasSuffix(name, ".jack") && !f.IsDir() {
			locs = append(locs, path.Join(fPath, name))
		}
	}
	return locs
}

func generate(loc string) error {
	trimmedName := strings.TrimSuffix(loc, ".jack")

	tokenFileName := trimmedName + "T.xml"
	tokenOut, err := os.Create(tokenFileName)
	defer tokenOut.Close()
	if err != nil {
		return err
	}

	codeOut, err := os.Create(trimmedName + ".vm")
	defer codeOut.Close()
	if err != nil {
		return err
	}

	b, err := os.ReadFile(loc)
	if err != nil {
		return err
	}

	t := NewTokenizer(tokenOut, b)
	if err := t.Tokenize(); err != nil {
		return err
	}

	p := NewParser(codeOut)
	if err := p.ReadTokenFile(tokenFileName); err != nil {
		return err
	}

	if err := p.Parse(); err != nil {
		return err
	}

	return nil
}
