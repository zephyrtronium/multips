package main

import "io"

type Write interface {
	Org() int64
	Len() int64
	Write(b io.WriterAt) error
	String() string
}

type Patch []Write
