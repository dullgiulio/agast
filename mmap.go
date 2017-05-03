package main

import (
	"errors"
	"os"
	"syscall"
)

type Mmap struct {
	data []byte
}

// TODO: Implement windows with mmap and others with readfull.

func NewMmap(f *os.File, fi os.FileInfo) (*Mmap, error) {
	size := fi.Size()
	if size == 0 {
		return nil, errors.New("file is empty")
	}
	if size < 0 {
		return nil, errors.New("file has negative size")
	}
	if size != int64(int(size)) {
		return nil, errors.New("file is too large")
	}
	m := &Mmap{}
	var err error
	m.data, err = syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Mmap) Data() []byte {
	return m.data
}

func (m *Mmap) Close() error {
	if m.data == nil {
		return nil
	}
	return syscall.Munmap(m.data)
}
