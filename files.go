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
	res   []resultGroup
}

func newFile(fname string, fi os.FileInfo) *file {
	return &file{fname: fname, fi: fi}
}

func (f *file) match(words [][]byte) bool {
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
	words   [][]byte
	types   ftypes
	results chan *file
	process chan *file
	done    chan struct{}
	wg      sync.WaitGroup
}

func newProc(nproc int, ts ftypes, words []string) *proc {
	ws := make([][]byte, len(words))
	for i := range words {
		ws[i] = []byte(words[i])
	}
	p := &proc{
		words:   ws,
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

func ellips(line string, n int) string {
	if len(line) <= n {
		return line
	}
	n = (n - 3) / 2
	return fmt.Sprintf("%s...%s", line[:n], line[n:])
}

// Run only one
func (p *proc) resulter() {
	var printed bool
	for f := range p.results {
		if f.err != nil {
			log.Printf("error: %s: %s", f.fname, f.err)
			continue
		}
		if printed {
			fmt.Print("\n")
		}
		printed = true
		fmt.Printf("\033[35m%s\033[0m\n", f.fname)
		for r := range f.res {
			for g := range f.res[r] {
				fmt.Printf("\033[32m%d\033[0m: ", f.res[r][g].num+1)
				line := f.res[r][g].line
				n := 0
				for _, hi := range f.res[r][g].hi {
					fmt.Print(ellips(line[n:hi.off], 50))
					n = hi.off + hi.n
					fmt.Printf("\033[91m%s\033[0m", line[hi.off:n])
				}
				fmt.Print(ellips(line[n:], 50))
				fmt.Print("\n")
			}
			if r < len(f.res)-1 {
				fmt.Printf("--\n")
			}
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
