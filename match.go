package main

import (
	"bytes"
)

type highlight struct {
	off int
	n   int
}

type resultGroup []result

type result struct {
	line string
	num  int
	hi   []highlight
}

type data struct {
	data  []byte
	words [][]byte
	nline int
	off   int
}

func (d *data) highlights(line []byte, start int, his []highlight, wn int) []highlight {
	for wn = wn + 1; wn < len(d.words); wn++ {
		pos := bytes.Index(line[start:], d.words[wn])
		if pos < 0 {
			break
		}
		his = append(his, highlight{pos + start, len(d.words[wn])})
		start = pos + len(d.words[wn])
	}
	return his
}

func (d *data) findWord(wn int) []result {
	if wn >= len(d.words) || d.off >= len(d.data) {
		return make([]result, 0, len(d.words))
	}
	pos := bytes.Index(d.data[d.off:], d.words[wn])
	if pos < 0 {
		return nil
	}
	var lineStart int
	pos = pos + d.off
	if d.nline < 0 {
		d.nline, lineStart = bytesBefore(d.data, pos, '\n')
	} else {
		newlines, _ := bytesBefore(d.data[d.off:], pos-d.off, '\n')
		lineStart = byteBefore(d.data, pos, '\n')
		d.nline = d.nline + newlines
	}

	nline := d.nline
	lineEnd := byteAfter(d.data, pos, '\n')
	line := d.data[lineStart:lineEnd]
	start := pos-lineStart
	his := []highlight{
		highlight{start, len(d.words[wn])},
	}

	// Get other matches in the same line, if present
	his = d.highlights(line, start, his, wn)
	d.off = lineEnd

	// Returns results in other lines or nil and the highlights in the same line
	res := d.findWord(wn+len(his))
	// Next word didn't match, unwind stack dropping partial results
	if res == nil {
		return nil
	}
	// By converting to string we copy the line as the underlying memory will be gone
	r := result{string(line), nline, his}
	res = append(res, r)
	return res
}

func (d *data) findWords(ws [][]byte) []resultGroup {
	var res []resultGroup
	d.words = ws
	d.nline = -1
	for {
		rs := d.findWord(0)
		if len(rs) == 0 {
			break
		}
		// Results are returned inverted because of recursive stacking
		for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
			rs[i], rs[j] = rs[j], rs[i]
		}
		res = append(res, rs)
	}
	return res
}

func bytesBefore(data []byte, off int, c byte) (int, int) {
	var cnt, last int
	for i := 0; i <= off; i++ {
		if data[i] == c {
			cnt++
			last = i+1
		}
	}
	return cnt, last
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
