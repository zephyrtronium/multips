package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	var destname, logname string
	var destf, logf *os.File
	var err error
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Remaining arguments are patches to apply.")
	}
	flag.StringVar(&destname, "a", "", "apply to given file (stdout if -)")
	flag.StringVar(&logname, "log", "-", "log to given file (stderr if -, none if empty)")
	flag.Parse()
	if destname == "" {
		log.Fatal("no target file given")
	}
	if destname == "-" {
		destf = os.Stdout
	} else {
		destf, err = os.OpenFile(destname, os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("unable to open %s for writing: %v", destname, err)
		}
	}
	if logname == "-" {
		logf = os.Stderr
	} else if logname == "" {
		logf = nil
	} else {
		logf, err = os.OpenFile(logname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("unable to open %s for logging: %v\nusing stderr instead", logname, err)
			logf = os.Stderr
		}
	}
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("no patches to apply")
	}
	patches := make([]Patch, len(args))
	for i, name := range args {
		if !strings.HasSuffix(strings.ToLower(name), ".ips") {
			fmt.Fprintf(logf, "Warning: %s doesn't look like an IPS patch. Continuing anyway.\n", name)
		}
		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(logf, "Warning: unable to open %s for reading (%v). Skipping.\n", name, err)
			continue
		}
		patches[i], err = ReadIPS(f)
		if err != nil {
			fmt.Fprintf(logf, "Warning: error while parsing %s: %v. Skipping.\n", name, err)
		}
		f.Close()
	}
	if len(patches) == 1 {
		fmt.Fprintf(logf, "Applying %s to %s\n", args[0], destname)
		Apply(patches[0], destf, logf)
		return
	}
	p := make([]Patch, 0, len(patches))
	names := make([]string, 0, len(patches))
	for i, v := range patches {
		if v != nil {
			p = append(p, v)
			names = append(names, args[i])
		}
	}
	if len(p) == 0 {
		log.Fatalf("no patches to apply")
	}
	howManyFuckingVariablesDoINeed := make(map[int]bool)
	for i := range p {
		for j := i+1; j < len(p); j++ {
			if c1, c2 := Conflict(p[i], p[j]); c1 != nil {
				fmt.Fprintf(logf, "CONFLICT between %s and %s:\n", names[i], names[j])
				for k := range c1 {
					fmt.Fprintf(logf, "\t%s\n\t%s\n", c1[k], c2[k])
				}
				howManyFuckingVariablesDoINeed[i] = true
				howManyFuckingVariablesDoINeed[j] = true
			}
		}
	}
	var patch Patch
	for i := range p {
		if !howManyFuckingVariablesDoINeed[i] {
			patch = p[i]
		}
	}
	if patch == nil {
		log.Fatal("all patches conflict!")
	}
	for i, v := range p {
		if !howManyFuckingVariablesDoINeed[i] {
			patch = Merge(patch, v)
		} else {
			fmt.Fprintf(logf, "SKIPPING %s due to conflict\n", names[i])
		}
	}
	Apply(patch, destf, logf)
}
