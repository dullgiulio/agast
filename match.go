package main

import (
	"bytes"
)

type result struct {
	line string
	num  int
}

type data struct {
	data []byte
	off  int
}

func (d *data) findWord(ws []string, wn int, nline int) ([]result, bool) {
	if wn >= len(ws) || d.off >= len(d.data) {
		return make([]result, 0, len(ws)), false
	}
	var sameline bool
	pos := bytes.Index(d.data[d.off:], []byte(ws[wn]))
	if pos < 0 {
		return nil, false
	}
	pos = pos + d.off
	lineStart := byteBefore(d.data, pos, '\n')
	lineEnd := byteAfter(d.data, pos, '\n')
	// note: line needs to be copied from the mmap memory, as it will be
	//       freed before being printed
	line := string(d.data[lineStart:lineEnd])
	if nline < 0 {
		nline = bytesBefore(d.data, pos, '\n')
	} else {
		newlines := bytesBefore(d.data[d.off:], pos-d.off, '\n')
		nline = nline + newlines
		if newlines == 0 {
			sameline = true
		}
	}
	d.off = pos + len(ws[wn])
	res, smline := d.findWord(ws, wn+1, nline)
	if !smline {
		if res == nil {
			return nil, false
		}
		res = append(res, result{line, nline})
	}
	return res, sameline
}

func (d *data) findWords(ws []string) []result {
	var res []result
	for {
		rs, _ := d.findWord(ws, 0, -1)
		if rs == nil {
			break
		}
		// Results are returned inverted because of recursive stacking
		for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
			rs[i], rs[j] = rs[j], rs[i]
		}
		if res == nil {
			res = rs
		} else {
			res = append(res, rs...)
		}
	}
	return res
}

func bytesBefore(data []byte, off int, c byte) int {
	var cnt int
	for i := 0; i < off; i++ {
		if data[i] == c {
			cnt++
		}
	}
	return cnt
}

func byteBefore(data []byte, pos int, c byte) int {
	for ; pos >= 0; pos-- {
		if data[pos] == c {
			return pos + 1
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
