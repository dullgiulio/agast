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
	dots    string
	types   ftypes
	cl      colorizer
	results chan *file
	process chan *file
	done    chan struct{}
	wg      sync.WaitGroup
}

func newProc(nproc int, ts ftypes, cl colorizer, words []string) *proc {
	ws := make([][]byte, len(words))
	for i := range words {
		ws[i] = []byte(words[i])
	}
	p := &proc{
		cl:      cl,
		words:   ws,
		types:   ts,
		done:    make(chan struct{}),
		results: make(chan *file), // Not buffered to give results ASAP
		process: make(chan *file, 2048),
	}
	p.dots = p.cl.colorize(hiEllipsis, "...") // do it once and for all
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

func (p *proc) ellips(line string, n int) string {
	if len(line) <= n {
		return line
	}
	n = (n - 3) / 2
	return fmt.Sprintf("%s%s%s", line[:n], p.dots, line[len(line)-n:])
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
		fmt.Println(p.cl.colorize(hiFilename, f.fname))
		for r := range f.res {
			for g := range f.res[r] {
				fmt.Print(p.cl.colorizef(hiNumber, "%d", f.res[r][g].num+1), ": ")
				line := f.res[r][g].line
				n := 0
				for _, hi := range f.res[r][g].hi {
					fmt.Print(p.ellips(line[n:hi.off], 160))
					n = hi.off + hi.n
					fmt.Print(p.cl.colorize(hiMatch, line[hi.off:n]))
				}
				fmt.Print(p.ellips(line[n:], 160))
				fmt.Print("\n")
			}
			if r < len(f.res)-1 {
				fmt.Println("--")
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
