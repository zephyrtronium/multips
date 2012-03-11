package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
)

type zpwrite struct {
	offset, repeat int64
	data []byte
}

func ReadZP(b io.Reader) (Patch, string, error) {
	if _, ok := b.(io.ByteReader); !ok {
		b = bufio.NewReader(b) // do this before gob.NewDecoder() does
	}
	p := make([]byte, 4)
	if n, err := b.Read(p); n < 4 || err != nil {
		return nil, "", err
	}
	if string(p) != "zp~\x7f" {
		return nil, "", errors.New("not a ZP patch")
	}
	decoder := gob.NewDecoder(b)
	metadata := ""
	if err := decoder.Decode(&metadata); err != nil {
		return nil, "", err
	}
	var patch Patch
	var absolute int64
	for {
		write := zpwrite{}
		var relative uint64
		if err := decoder.Decode(&relative); err != nil {
			return nil, "", err
		}
		absolute += int64(relative)
		write.offset = absolute
		var length int
		if err := decoder.Decode(&write.data); err != nil {
			return nil, "", err
		}
		if err := decoder.Decode(&write.repeat); err != nil {
			return nil, "", err
		}
		if relative == 0 && len(write.data) == 0 && write.repeat == 0 {
			return patch, metadata, nil
		}
		write.data = make([]byte, length)
	}
	return nil, "", nil // unreachable
}

func WriteZP(b io.Writer, patch Patch, metadata string) (err error) {
	encoder := gob.NewEncoder(b)
	if _, err = b.Write([]byte("zp~\x7f")); err != nil {
		return
	}
	if err = encoder.Encode(metadata); err != nil {
		return
	}
	var relative int64
	for _, write := range patch {
		if err = encoder.Encode(uint64(write.Org()-relative)); err != nil {
			return
		}
		switch w := write.(type) {
		case *zpwrite:
			if err = encoder.Encode(w.data); err != nil {
				return
			}
			if err = encoder.Encode(uint64(w.repeat)); err != nil {
				return
			}
		case *ipswrite:
			if err = encoder.Encode(w.data); err != nil {
				return
			}
			if err = encoder.Encode(1); err != nil {
				return
			}
		case *rlewrite:
			if err = encoder.Encode([]byte{w.data}); err != nil {
				return
			}
			if err = encoder.Encode(w.num); err != nil {
				return
			}
		default:
			panic("write type not implemented")
		}
		relative = write.Org() + write.Len()
	}
	_, err = b.Write([]byte{0,0,0})
	return
}

func (self *zpwrite) Org() int64 {
	return self.offset
}

func (self *zpwrite) Len() int64 {
	return self.repeat * int64(len(self.data))
}

func (self *zpwrite) Write(b io.WriterAt) (err error) {
	if self.repeat == 1 {
		_, err = b.WriteAt(self.data, self.Org())
		return
	}
	l := self.Len()
	if l < 1 << 20 { // just some arbitrary limit i guess
		// Since the write is small, just do it all at once.
		// This is probably suboptimal if the writerat isn't a file,
		// but it's probably a file.
		w := make([]byte, l)
		for i := 0; int64(i) < l; i += len(self.data) {
			copy(w[i:], self.data)
		}
		_, err = b.WriteAt(w, self.Org())
		return
	}
	for i := int64(0); i < l; i += int64(len(self.data)) {
		_, err = b.WriteAt(self.data, self.Org() + i)
		if err != nil {
			return
		}
	}
	return
}

func (self *zpwrite) String() string {
	if self.repeat == 1 {
		return fmt.Sprintf("ZP data: %d bytes written to %#x", len(self.data), self.Org())
	}
	return fmt.Sprintf("ZP data: %d bytes written to %#x (%d bytes x%d)", self.Len(), self.Org(), len(self.data), self.repeat)
}
