package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ftypes []string

func (ft ftypes) valid(fname string) bool {
	for n := range ft {
		if strings.HasSuffix(fname, ft[n]) {
			return true
		}
	}
	return false
}

type proc struct {
	types ftypes
}

func (p *proc) process(filename string, fi os.FileInfo) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open: %s", err)
	}
	defer f.Close()
	mmap, err := NewMmap(f, fi)
	if err != nil {
		return fmt.Errorf("cannot mmap: %s", err)
	}
	defer mmap.Close()
	// 1. search for words (TODO: case insensitive search)
	// 2. when found, tokenize and give score
	fmt.Printf("%s", mmap.Data())
	return nil
}

func (p *proc) walk(path string, fi os.FileInfo, err error) error {
	// TODO: Improve this
	if err != nil {
		return err
	}
	if path[0] == '.' && len(path) > 1 {
		return filepath.SkipDir
	}
	if p.types.valid(path) {
		if err = p.process(path, fi); err != nil {
			fmt.Printf("%s: cannot process: %s\n", path, err)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	dir := flag.Arg(0)
	if dir == "" {
		log.Fatal("You need to specify a directory")
	}
	p := &proc{
		types: ftypes([]string{".go", ".php"}),
	}
	if err := filepath.Walk(dir, p.walk); err != nil {
		log.Fatal("cannot walk: ", err)
	}
}
