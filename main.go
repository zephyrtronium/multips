package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	var destname, logname, pdestname, metadname, metadata string
	var destf, logf, pdestf *os.File
	var err error
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Remaining arguments are patches to apply.")
	}
	flag.StringVar(&destname, "a", "", "apply to given file")
	flag.StringVar(&pdestname, "merge", "", "write merged patch in ZP format to given file (stdout if -)")
	flag.StringVar(&logname, "log", "-", "log to given file (stderr if -, none if empty)")
	flag.StringVar(&metadname, "meta", "", "use contents of given file as ZP output metadata (stdin if -)")
	flag.Parse()
	if destname == "" && pdestname == "" {
		log.Fatal("no target file given")
	}
	if destname != "" {
		destf, err = os.OpenFile(destname, os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("unable to open %s for writing: %v", destname, err)
		}
	}
	if pdestname == "-" {
		pdestf = os.Stdout
	} else if pdestname != "" {
		pdestf, err = os.Create(pdestname)
		if err != nil {
			log.Fatalf("unable to create %s: %v", pdestname, err)
		}
	}
	if logname == "-" {
		logf = os.Stderr
	} else if logname == "" {
		logf = nil
	} else {
		logf, err = os.OpenFile(logname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("unable to open %s for logging: %v\nusing stderr instead\n", logname, err)
			logf = os.Stderr
		}
	}
	if metadname == "-" {
		m, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(logf, "error reading metadata from stdin:", err)
		}
		metadata = string(m)
	} else if metadname != "" {
		m, err := ioutil.ReadFile(metadname)
		if err != nil {
			fmt.Fprintf(logf, "error reading metadata from %s: %v\n", metadname, err)
		}
		metadata = string(m)
	}
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("no patches to apply")
	}
	patches := make([]Patch, len(args))
	for i, name := range args {
		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(logf, "Warning: unable to open %s for reading (%v). Skipping.\n", name, err)
			f.Close()
			continue
		}
		if strings.HasSuffix(strings.ToLower(name), ".ips") {
			patches[i], err = ReadIPS(f)
			if err != nil {
				fmt.Fprintf(logf, "Warning: error while parsing %s: %v. Skipping.\n", name, err)
			}
		} else if strings.HasSuffix(strings.ToLower(name), ".zp") {
			meta := ""
			patches[i], meta, err = ReadZP(f)
			if err != nil {
				fmt.Fprintf(logf, "Warning: error while parsing %s: %v. Skipping.\n", name, err)
			} else {
				fmt.Fprintln(logf, meta)
			}
		} else {
			meta := ""
			patches[i], meta, err = ReadZP(f)
			if err != nil {
				fmt.Fprintf(logf, "Warning: error while parsing %s as ZP: %v. Trying IPS.\n", name, err)
				f.Seek(0, 0)
				patches[i], err = ReadIPS(f)
				if err != nil {
					fmt.Fprintf(logf, "Warning: error while parsing %s as IPS: %v. Skipping.\n", name, err)
				}
			} else {
				fmt.Fprintln(logf, meta)
			}
		}
		f.Close()
	}
	if len(patches) == 1 {
		if pdestf != nil {
			if err = WriteZP(pdestf, patches[0], metadata); err != nil {
				fmt.Fprintf(logf, "WARNING: error writing patch to %s: %v", pdestname, err)
			}
		}
		if destf != nil {
			fmt.Fprintf(logf, "Applying %s to %s\n", args[0], destname)
			Apply(patches[0], destf, logf)
		}
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
	for i, v := range p[1:] {
		if !howManyFuckingVariablesDoINeed[i] {
			patch = Merge(patch, v)
		} else {
			fmt.Fprintf(logf, "SKIPPING %s due to conflict\n", names[i+1])
		}
	}
	if pdestf != nil {
		if err = WriteZP(pdestf, patch, metadata); err != nil {
			fmt.Fprintf(logf, "WARNING: error writing patch to %s: %v", pdestname, err)
		}
	}
	if destf != nil {
		if err = Apply(patch, destf, logf); err != nil {
			fmt.Fprintf(logf, "ERROR applying patch to %s: %v", destname, err)
		}
	}
}
