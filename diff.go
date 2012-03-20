package main

import (
	"fmt"
	"io"
)

type diffwrite struct {
	offset int64
	data   []byte
}

func (self *diffwrite) Org() int64 {
	return self.offset
}

func (self *diffwrite) Len() int64 {
	return int64(len(self.data))
}

func (self *diffwrite) Write(b io.WriterAt) (err error) {
	_, err = b.WriteAt(self.data, self.Org())
	return
}

func (self *diffwrite) String() string {
	return fmt.Sprintf("Diff: %d bytes written to %#x", self.Len(), self.Org())
}

func Diff(fs []io.ByteReader) (patch Patch, conflict int64, err error) {
	conflict = -1
	buf := make([]byte, len(fs))
	done := make([]bool, len(fs))
	anydone := false
	var curdiff *diffwrite
	diffing := false
	var offset int64
	for {
		for i, f := range fs {
			if done[i] {
				continue
			}
			if buf[i], err = f.ReadByte(); err != nil {
				if err != io.EOF {
					return
				}
				done[i] = true
				anydone = true
			}
		}
		set := make(map[byte]bool)
		var vals []byte
		for i, c := range buf {
			if !done[i] {
				set[c] = true
				vals = append(vals, c)
			}
		}
		switch len(set) {
		case 0: // all files done
			if diffing {
				patch = append(patch, curdiff)
			}
			err = nil
			return
		case 1: // all files the same, except those done
			if anydone { // the byte is a diff
				if diffing {
					curdiff.data = append(curdiff.data, vals[0])
				} else {
					curdiff = &diffwrite{offset, []byte{vals[0]}}
					diffing = true
				}
			} else if diffing { // all bytes are the same; diff is done
				patch = append(patch, curdiff)
				diffing = false
			} // else case is handled in anydone
		case 2: // two files differ
			if diffing {
				curdiff.data = append(curdiff.data, vals[1])
			} else {
				curdiff = &diffwrite{offset, []byte{vals[1]}}
				diffing = true
			}
		default: // diffs between multiple files at one offset
			conflict = offset
			return
		}
		offset++
	}
	panic("unreachable")
}
