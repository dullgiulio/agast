package main

import (
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

type file struct {
	fi    os.FileInfo
	fname string
	err   error
	res   []result
}

func newFile(fname string, fi os.FileInfo) *file {
	return &file{fname: fname, fi: fi}
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
	d := data{data: mmap.Data()}
	f.res = d.findWords(words)
	return f.res != nil
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
		fmt.Print("\n")
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
