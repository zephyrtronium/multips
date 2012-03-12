package main

import "io"

func lg(x uint64) int {
	var i int
	for i = 0; x != 0; i++ {
		x >>= 1
	}
	return i-1
}

func VLIEncode(x uint64) []byte {
	if x < 128 {
		return []byte{byte(x)}
	}
	b := make([]byte, (lg(x) - 1) >> 3 + 2)
	for i := len(b)-1; i > 0; i-- {
		b[i] = byte(x)
		x >>= 8
	}
	b[0] = ^byte(len(b)) + 2
	return b
}

func VLIStreamOut(b io.Writer, x uint64) (int, error) {
	return b.Write(VLIEncode(x))
}

func VLIStreamIn(b io.Reader) (x uint64, err error) {
	p := make([]byte, 1)
	if _, err = b.Read(p); err != nil {
		return
	}
	if p[0] < 128 {
		x = uint64(p[0])
		return
	}
	n := -int(int8(p[0]))
	p = make([]byte, n)
	if _, err = b.Read(p); err != nil {
		if err == io.EOF {
			err = nil
		} else {
			return
		}
	}
	for _, v := range p {
		x = x << 8 | uint64(v)
	}
	return
}

func VLIStreamOutBytes(b io.Writer, s []byte) (int, error) {
	var n int
	var err error
	if n, err = VLIStreamOut(b, uint64(len(s))); err != nil {
		return n, err
	}
	m := n
	if n, err = b.Write(s); err != nil {
		return m + n, err
	}
	return m + n, nil
}

func VLIStreamInBytes(b io.Reader) (x []byte, err error) {
	var n uint64
	if n, err = VLIStreamIn(b); err != nil {
		return
	}
	x = make([]byte, int(n))
	_, err = b.Read(x)
	if err == io.EOF {
		err = nil
	}
	return
}
