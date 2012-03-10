package main

import (
	"errors"
	"fmt"
	"io"
)

type ipswrite struct {
	start uint32
	data []byte
}

type rlewrite struct {
	start uint32
	num uint16
	data byte
}

func add(patch Patch, write Write) Patch {
	// The approach here is Θ(n) instead of optimal O(lg(n)), but if the
	// patch is written in sorted order, then its actual time is constant.
	for i := len(patch)-1; i >= 0; i-- {
		if patch[i].Org() < write.Org() {
			return append(patch[:i], append(Patch{write}, patch[i:]...)...)
		}
	}
	return append(Patch{write}, patch...)
}

func ReadIPS(b io.Reader) (Patch, error) {
	p := make([]byte, 65536) // large enough to hold any single write
	n, err := b.Read(p[:5])
	if n < 5 || err != nil {
		return nil, err
	}
	if string(p[:5]) != "PATCH" {
		return nil, errors.New("not an IPS patch")
	}
	patch := make(Patch, 0)
	for {
		n, err = b.Read(p[:3])
		if n < 3 || err != nil && err != io.EOF {
			return nil, err
		}
		org := uint32(p[0]) << 16 | uint32(p[1]) << 8 | uint32(p[2])
		if org == 0x454f46 { // EOF
			return patch, nil
		}
		if err != nil {
			return nil, err
		}
		n, err = b.Read(p[:2])
		if n < 2 || err != nil {
			return nil, err
		}
		num := uint16(p[0]) << 8 | uint16(p[1])
		if num == 0 { // RLE
			n, err = b.Read(p[:2])
			if n < 2 || err != nil {
				return nil, err
			}
			num = uint16(p[0]) << 8 | uint16(p[1])
			n, err = b.Read(p[:1])
			if n < 1 || err != nil {
				return nil, err
			}
			patch = add(patch, &rlewrite{org, num, p[0]})
		} else {
			n, err = b.Read(p[:num])
			if n < int(num) || err != nil {
				return nil, err
			}
			data := make([]byte, num)
			copy(data, p[:num])
			patch = add(patch, &ipswrite{org, data})
		}
	}
	return nil, nil
}

func (self *ipswrite) Org() uint64 {
	return uint64(self.start)
}

func (self *rlewrite) Org() uint64 {
	return uint64(self.start)
}

func (self *ipswrite) Len() uint64 {
	return uint64(len(self.data))
}

func (self *rlewrite) Len() uint64 {
	return uint64(self.num)
}

func (self *ipswrite) Write(b io.WriterAt) (err error) {
	_, err = b.WriteAt(self.data, int64(self.start))
	return
}

func (self *rlewrite) Write(b io.WriterAt) (err error) {
	s := make([]byte, self.num)
	for i := range s {
		s[i] = self.data
	}
	_, err = b.WriteAt(s, int64(self.start))
	return
}

func (self *ipswrite) String() string {
	return fmt.Sprintf("IPS data: %d bytes written to %x", self.Len(), self.Org())
}

func (self *rlewrite) String() string {
	return fmt.Sprintf("IPS RLE: %x written to %x %d times", self.data, self.Org(), self.Len())
}
