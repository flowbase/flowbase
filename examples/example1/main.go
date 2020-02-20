package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	BUFSIZE = 16
)

func main() {
	lines := ReadFileByLine("filelist.txt")
	lines = Grep(lines, "boot")
	for line := range lines {
		if line != "" {
			fmt.Println(line)
		}
	}
}

// --------------------------------------------------------------------------------
// Components
// --------------------------------------------------------------------------------

func ReadFileByLine(fileName string) (lines chan string) {
	lines = make(chan string, BUFSIZE)
	go func() {
		defer close(lines)

		file, err := os.Open(fileName)
		if err != nil {
			log.Fatalln(err)
		}
		defer file.Close()

		sc := bufio.NewScanner(file)
		for sc.Scan() {
			if err := sc.Err(); err != nil {
				log.Fatal(err)
			}
			lines <- sc.Text()
		}
	}()
	return lines
}

func Map(inlines chan string, fn func(ins string) (outs string)) (outlines chan string) {
	outlines = make(chan string, BUFSIZE)
	go func() {
		defer close(outlines)
		for l := range inlines {
			outlines <- fn(l)
		}
	}()
	return outlines
}

// --------------------------------------------------------------------------------
// Grepper
// --------------------------------------------------------------------------------

func Grep(inl chan string, pattern string) (outl chan string) {
	g := NewGrepper(pattern)
	g.InLines = inl
	go g.Run()
	return g.OutLines
}

type Grepper struct {
	InLines  chan string
	OutLines chan string
	pattern  string
}

func NewGrepper(pattern string) *Grepper {
	return &Grepper{
		InLines:  make(chan string, BUFSIZE),
		OutLines: make(chan string, BUFSIZE),
		pattern:  pattern,
	}
}

func (p *Grepper) Run() {
	defer close(p.OutLines)
	for l := range p.InLines {
		if strings.Contains(l, p.pattern) {
			p.OutLines <- l
		}
	}
}
