package main

import (
	"fmt"
	"io"
)

func Apply(patch Patch, b io.WriterAt, log io.Writer) {
	for _, write := range patch {
		if log != nil {
			fmt.Fprintln(log, write)
		}
		if err := write.Write(b); err != nil {
			if log != nil {
				fmt.Fprintf(log, "XXX ERROR: %v", err)
			}
			return
		}
	}
	if log != nil {
		fmt.Fprintf(log, "Patching finished.")
	}
}
