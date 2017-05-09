package main

import (
	"bytes"
)

type highlight struct {
	off int
	n   int
}

type result struct {
	line string
	num  int
	hi   []highlight
}

type data struct {
	data  []byte
	nline int
	off   int
}

func (d *data) findWord(ws []string, wn int) ([]result, []highlight) {
	if wn >= len(ws) || d.off >= len(d.data) {
		return make([]result, 0, len(ws)), nil
	}
	pos := bytes.Index(d.data[d.off:], []byte(ws[wn]))
	if pos < 0 {
		return nil, nil
	}
	var sameline bool
	pos = pos + d.off
	if d.nline < 0 {
		d.nline = bytesBefore(d.data, pos, '\n')
	} else {
		newlines := bytesBefore(d.data[d.off:], pos-d.off, '\n')
		d.nline = d.nline + newlines
		if newlines == 0 {
			sameline = true
		}
	}

	// TODO: extract line only if not sameline
	lineStart := byteBefore(d.data, pos, '\n')
	lineEnd := byteAfter(d.data, pos, '\n')
	// note: line needs to be copied from the mmap memory, as it will be
	//       freed before being printed
	line := string(d.data[lineStart:lineEnd])

	d.off = pos + len(ws[wn])
	// Returns results in other lines or nil and the highlights in the same line
	res, his := d.findWord(ws, wn+1)
	// Next word didn't match, unwind stack dropping partial results
	if res == nil && his == nil {
		return nil, nil
	}
	hi := highlight{pos - lineStart, len(ws[wn])}
	if sameline {
		his = append([]highlight{hi}, his...)
		return res, his
	}
	if his == nil {
		his = []highlight{hi}
	} else {
		his = append([]highlight{hi}, his...)
	}
	r := result{line, d.nline, his}
	res = append(res, r)
	return res, nil
}

func (d *data) findWords(ws []string) []result {
	var res []result
	d.nline = -1
	for {
		rs, _ := d.findWord(ws, 0)
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
			// TODO: Skip one result if on the same line as the last known
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
