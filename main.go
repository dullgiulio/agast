package main

import (
	"flag"
	"log"
	"os"
)

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
	exts := ftypes([]string{".go", ".php", ".js", ".css", ".html"})
	cl := rgcolors{} // TODO: only if terminal supports it
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
	p := newProc(nproc, exts, cl, words)
	go p.resulter()
	if err := p.run(dir); err != nil {
		log.Printf("error: %s", err)
	}
	p.wait()
}
