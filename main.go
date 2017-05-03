package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type result struct {
	line string
	num  int
}

type file struct {
	fi    os.FileInfo
	fname string
	err   error
	res   []result
}

func newFile(fname string, fi os.FileInfo) *file {
	return &file{fname: fname, fi: fi}
}

func bytesBefore(data []byte, c byte) int {
	var cnt int
	for i := 0; i < len(data); i++ {
		if data[i] == c {
			cnt++
		}
	}
	return cnt
}

func byteBefore(data []byte, pos int, c byte) int {
	if pos > len(data) {
		log.Fatalf("invalid call")
	}
	for ; pos >= 0; pos-- {
		if data[pos] == c {
			return pos
		}
	}
	return 0
}

func byteAfter(data []byte, pos int, c byte) int {
	for ; pos < len(data); pos++ {
		if data[pos] == c {
			return pos
		}
	}
	return len(data)
}

type submatch struct {
	// Offsets of words found
	woffs []int
	end   int
}

func (s *submatch) seq(data []byte, words []string, off int) bool {
	offset := off
	for wn := range words {
		pos := bytes.Index(data[offset:], []byte(words[wn]))
		if pos < 0 {
			return false
		}
		offset = offset + pos
		s.woffs = append(s.woffs, offset)
		s.end = offset + off + len(words[wn])
	}
	return true
}

func (f *file) match(words []string) bool {
	fh, err := os.Open(f.fname)
	if err != nil {
		f.err = fmt.Errorf("cannot open: %s", err)
		return false
	}
	defer fh.Close()
	mmap, err := NewMmap(fh, f.fi)
	if err != nil {
		f.err = fmt.Errorf("cannot mmap: %s", err)
		return false
	}
	defer mmap.Close()
	data := mmap.Data()
	// 1. search for words (TODO: case insensitive search)
	offset := 0
	matched := false
	for {
		s := &submatch{}
		if !s.seq(data, words, offset) {
			// No more matches in this file
			break
		}
		offset = s.end
		matched = true
		if offset >= len(data) {
			break
		}
		for n := range s.woffs {
			lineStart := byteBefore(data, s.woffs[n], '\n')
			lineEnd := byteAfter(data, s.woffs[n], '\n')
			// note: line needs to be copied from the mmap memory, as it will be
			//		 freed before being printed
			line := string(data[lineStart+1 : lineEnd])
			nline := bytesBefore(data[0:s.woffs[n]], '\n')
			f.res = append(f.res, result{line: line, num: nline})
		}
	}
	return matched
}

type proc struct {
	words   []string
	types   ftypes
	results chan *file
	process chan *file
	done    chan struct{}
	wg      sync.WaitGroup
}

func newProc(nproc int, ts ftypes, words []string) *proc {
	p := &proc{
		words:   words,
		types:   ts,
		done:    make(chan struct{}),
		results: make(chan *file), // Not buffered to give results ASAP
		process: make(chan *file, 2048),
	}
	p.wg.Add(nproc)
	for i := 0; i < nproc; i++ {
		go p.processor()
	}
	return p
}

func (p *proc) wait() {
	p.wg.Wait()
	close(p.results)
	<-p.done
}

// Run only one
func (p *proc) resulter() {
	for f := range p.results {
		if f.err != nil {
			log.Printf("error: %s: %s", f.fname, f.err)
			continue
		}
		fmt.Printf("%s:\n", f.fname)
		for r := range f.res {
			fmt.Printf("%d: %s\n", f.res[r].num, f.res[r].line)
		}
	}
	p.done <- struct{}{}
}

func (p *proc) processor() {
	for f := range p.process {
		ok := f.match(p.words)
		// Matches and errors are forwarded to output
		if ok || f.err != nil {
			p.results <- f
		}
	}
	p.wg.Done()
}

func (p *proc) file(path string, fi os.FileInfo, err error) error {
	// TODO: Improve this
	if err != nil {
		return err
	}
	if path[0] == '.' && len(path) > 1 {
		return filepath.SkipDir
	}
	// TODO: check that it's a file
	if p.types.valid(path) {
		p.process <- newFile(path, fi)
	}
	return nil
}

// TODO: Support multiple dirs
func (p *proc) run(dir string) error {
	err := filepath.Walk(dir, p.file)
	close(p.process)
	return err
}

func isDir(d string) bool {
	fi, err := os.Stat(d)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return fi.IsDir()
}

func main() {
	nproc := 4
	flag.Parse()
	dir := "."
	words := flag.Args()
	if flag.NArg() > 0 {
		last := flag.Arg(flag.NArg() - 1)
		if last != "" && isDir(last) {
			dir = last
			words = words[0 : len(words)-1]
		}
	}
	if dir == "" {
		log.Fatal("You need to specify a directory")
	}
	p := newProc(nproc, ftypes([]string{".php"}), words)
	go p.resulter()
	if err := p.run(dir); err != nil {
		log.Printf("error: %s", err)
	}
	p.wait()
}
