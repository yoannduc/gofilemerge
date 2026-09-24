package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/yoannduc/gofilemerge/pkg/merger"
)

var (
	errFileNotGoFile = errors.New("file is not go file")
)

// Flag vars.
var (
	shouldDel bool
	outPath   string
	pkgName   string
)

func main() {
	// Remove date & time from std logs.
	log.SetFlags(0)
	flag.BoolVar(&shouldDel, "rm", false, "remove merged files")
	flag.StringVar(&outPath, "out", "", "output file; defaults to stdout")
	flag.StringVar(&pkgName, "pkg", "", "output package name; defaults to first package name scanned")
	flag.Usage = func() {
		fmt.Println("usage: gofilemerge [flags] [path ...]")
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		return
	}

	m := merger.NewMerger()
	if pkgName != "" {
		m.SetPackage(pkgName)
	}

	for _, arg := range args {
		if !isGoFilename(arg) {
			log.Fatal(fmt.Errorf(`file "%s": %w`, arg, errFileNotGoFile))
		}

		b, err := os.ReadFile(arg)
		if err != nil {
			log.Fatal(fmt.Errorf(`file "%s": %w`, arg, err))
		}
		_, err = m.Write(b)
		if err != nil {
			log.Fatal(fmt.Errorf(`file "%s": %w`, arg, err))
		}

		if shouldDel {
			defer os.Remove(arg)
		}
	}

	if outPath == "" {
		_, err := m.WriteTo(os.Stdout)
		if err != nil {
			log.Fatal(fmt.Errorf(`output: %w`, err))
		}
		return
	}

	if !isGoFilename(outPath) {
		log.Fatal(fmt.Errorf(`output "%s": %w`, outPath, errFileNotGoFile))
	}

	f, err := os.Create(outPath)
	if err != nil {
		log.Fatal(fmt.Errorf(`output "%s": %w`, outPath, err))
	}
	defer f.Close()
	_, err = m.WriteTo(f)
	if err != nil {
		log.Fatal(fmt.Errorf(`output "%s": %w`, outPath, err))
	}
}

func isGoFilename(name string) bool {
	return strings.HasSuffix(name, ".go")
}
