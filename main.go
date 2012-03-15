package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type argd struct {
	patch          Patch
	name, metadata string
	conflicts      bool
}

func fatalf(format string, vals ...interface{}) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format, vals...)
	fmt.Fprint(os.Stderr, "\n")
	os.Exit(1)
	panic("unreachable")
}

func fatal(stuff ...interface{}) {
	fmt.Fprintln(os.Stderr, append([]interface{}{"FATAL:"}, stuff...)...)
	os.Exit(1)
	panic("unreachable")
}

func readarg(patches []argd, arg string, logf *os.File) []argd {
	var f *os.File
	var err error
	list := strings.Split(arg, ",") // files to be diffed are split by commas
	if len(list) == 1 {
		if f, err = os.Open(arg); err != nil {
			fmt.Fprintf(logf, "WARNING: Failed to open %s: %v. SKIPPING.\n", arg, err)
			return patches // file probably doesn't exist or user lacks read rights
		}
		var t Patch
		// try IPS first because it's more common
		if t, err = ReadIPS(f); err == BadIPS {
			// failed to read as IPS; try ZP
			f.Seek(0, 0)
			var meta string
			if t, meta, err = ReadZP(f); err == BadZP {
				fmt.Fprintln(logf, "WARNING: Failed to read", arg, "as a patch. SKIPPING.")
				return patches
			} else if err != nil {
				fmt.Fprintf(logf, "WARNING: Failed to read %s: %v. SKIPPING.\n", arg, err)
				return patches
			} else {
				fmt.Fprintln(logf, "Read", arg, "as ZP.")
				return append(patches, argd{t, arg, meta, false})
			}
		} else if err != nil { // IPS read failed for some reason
			fmt.Fprintf(logf, "WARNING: Failed to read %s: %v. SKIPPING.\n", arg, err)
			return patches
		} else {
			fmt.Fprintln(logf, "Read", arg, "as IPS.")
			return append(patches, argd{t, arg, "", false})
		}
	} else { // multiple files to be diffed
		var todiff []io.ByteReader
		for _, name := range list {
			if f, err = os.Open(name); err != nil {
				fmt.Fprintf(logf, "WARNING: Failed to open %s: %v. SKIPPING diff of %v.\n", name, err, list)
				return patches
			}
			todiff = append(todiff, bufio.NewReader(f))
		}
		if t, conflict, err := Diff(todiff); err != nil {
			fmt.Fprintf(logf, "WARNING: Failed to diff %v: %v. SKIPPING.\n", list, err)
			return patches
		} else if conflict >= 0 {
			fmt.Fprintf(logf, "WARNING: Diff of %v conflicts (first) at %d. SKIPPING.\n", list, conflict)
			return patches
		} else {
			fmt.Fprintln(logf, "Diff of", list, "successful.")
			return append(patches, argd{t, fmt.Sprint(list), "", false})
		}
	}
	panic("unreachable")
}

func main() {
	var patches []argd
	var destname, patchname, metaname, logname string
	var metadata []byte
	var destf, patchf, logf *os.File
	var err error
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "After these flags, arguments containing , are treated as lists of files to diff,")
		fmt.Fprintln(os.Stderr, "with the first file being considered the old and all others the new.")
		fmt.Fprintln(os.Stderr, "Other arguments are read as patches. IPS and ZP formats are supported.")
	}
	flag.StringVar(&destname, "a", "", "apply to given file")
	flag.StringVar(&patchname, "merge", "", "write merged patch in ZP format to given file (stdout if -)")
	flag.StringVar(&metaname, "meta", "", "use contents of given file as ZP output metadata (stdin if -)")
	flag.StringVar(&logname, "log", "-", "log to given file (stderr if - or empty)")
	flag.Parse()
	if destname == "" && patchname == "" {
		fatal("no output files")
	}
	if destname != "" {
		if destf, err = os.OpenFile(destname, os.O_WRONLY, 0644); err != nil {
			fatal(err)
		}
	}
	if patchname != "" {
		if patchname == "-" {
			patchf = os.Stdout
		} else if patchf, err = os.Create(patchname); err != nil {
			fatal(err)
		}
	}
	if metaname != "" {
		if metaname == "-" {
			if metadata, err = ioutil.ReadAll(os.Stdin); err != nil { // can this happen?
				fatal(err)
			}
		} else if metadata, err = ioutil.ReadFile(metaname); err != nil {
			fatal(err)
		}
	}
	if logname == "-" || logname == "" {
		logf = os.Stderr
	} else if logf, err = os.OpenFile(destname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
		fatal(err)
	}
	args := flag.Args()
	for _, arg := range args {
		patches = readarg(patches, arg, logf)
	}
	switch len(patches) {
	case 0:
		fatal("nothing to do")
	case 1:
		patch := patches[0]
		if patchf != nil {
			fmt.Fprintln(logf, "Writing patch...")
			if err = WriteZP(patchf, patch.patch, metadata); err != nil {
				fmt.Fprintln(logf, "ERROR writing patch:", err)
			}
		}
		if destf != nil {
			fmt.Fprintln(logf, "Applying", patch.name, "to", destname)
			if err = Apply(destf, patch.patch, logf); err != nil {
				fmt.Fprintln(logf, "ERROR applying patch:", err)
			}
		}
	default:
		fmt.Fprintln(logf, "Testing for conflicts between", len(patches), "patches...")
		// first find conflicts
		for i := range patches {
			fmt.Fprintln(logf, patches[i].name)
			for j := i + 1; j < len(patches); j++ {
				fmt.Fprint(logf, "  ", patches[j].name, ": ")
				if c1, _ := Conflict(patches[i].patch, patches[j].patch); c1 != nil {
					fmt.Fprintln(logf, "CONFLICT. SKIPPING.")
					patches[i].conflicts = true
					patches[j].conflicts = true
				} else {
					fmt.Fprintln(logf, "ok")
				}
			}
		}
		index := -1
		for i, p := range patches {
			if !p.conflicts {
				index = i
			}
		}
		if index < 0 {
			fatal("all files conflict")
		}
		patch := patches[index].patch
		for i, p := range patches {
			if !p.conflicts && i != index {
				patch = Merge(patch, p.patch)
			}
		}
		if patchf != nil {
			fmt.Fprintln(logf, "Writing patch to", patchname)
			if err = WriteZP(patchf, patch, metadata); err != nil {
				fmt.Fprintln(logf, "ERROR writing patch:", err)
			}
		}
		if destf != nil {
			fmt.Fprintln(logf, "Applying patch to", destname)
			if err = Apply(destf, patch, logf); err != nil {
				fmt.Fprintln(logf, "ERROR applying patch:", err)
			}
		}
	}
}
