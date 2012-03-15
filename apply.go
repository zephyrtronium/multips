package main

import (
	"fmt"
	"io"
)

func Apply(b io.WriterAt, patch Patch, log io.Writer) (err error) {
	fmt.Fprintf(log, "%d total writes\n", len(patch))
	for i, write := range patch {
		if write == nil {
			continue // never should happen, but doesn't hurt to be safe
		}
		fmt.Fprintf(log, "Write %d (%.0f%%):\n", i+1, float32(i+1)/float32(len(patch)))
		fmt.Fprintln(log, write)
		if err = write.Write(b); err != nil {
			fmt.Fprintf(log, "XXX ERROR: %v\n", err)
			return
		}
	}
	fmt.Fprintln(log, "Patching finished.")
	return
}
