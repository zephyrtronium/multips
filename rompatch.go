package main

import "io"

type Write interface {
	Org() uint64
	Len() uint64
	Write(b io.WriterAt) error
	String() string
}

type Patch []Write
