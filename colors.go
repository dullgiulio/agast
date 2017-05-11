package main

import (
	"fmt"
)

type hielement int

const (
	hiNone hielement = iota
	hiFilename
	hiNumber
	hiEllipsis
	hiMatch
)

type colorizer interface {
	colorize(hielement, string) string
	colorizef(hielement, string, ...interface{}) string
}

type nocolor struct{}

func (nocolor) colorize(_ hielement, s string) string {
	return s
}

func (nocolor) colorizef(_ hielement, frmt string, data ...interface{}) string {
	return fmt.Sprintf(frmt, data...)
}

// ripgrep color scheme
type rgcolors struct{}

func (rgcolors) code(he hielement) string {
	// TODO: move these to constants
	switch he {
	case hiFilename:
		return "\033[35m"
	case hiNumber:
		return "\033[32m"
	case hiEllipsis:
		return "\033[1;34m"
	case hiMatch:
		return "\033[91m"
	}
	return ""
}

func (r rgcolors) colorize(he hielement, s string) string {
	if he == hiNone {
		return s
	}
	return r.code(he) + s + "\033[0m"
}

func (r rgcolors) colorizef(he hielement, frmt string, data ...interface{}) string {
	if he == hiNone {
		return fmt.Sprintf(frmt, data...)
	}
	return fmt.Sprintf(r.code(he)+frmt+"\033[0m", data...)
}
