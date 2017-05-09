package main

import (
	"fmt"
	"testing"
)

const data0 = `00 Hello
01 World how is
02 life or something going
03
04 Hello
05 World how is
06 life or something going
07`

func TestMatchWords(t *testing.T) {
	d := data{data: []byte(data0)}
	words := []string{"Hello", "how", "life", "going"}
	res := d.findWords(words)
	fmt.Printf("%s\n%+v\n%+v\n", data0, words, res)
}
