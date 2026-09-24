package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"log"
	"maps"
	"os"
	"time"
)

type imprt struct {
	Name string
	Path string
}

type imprtHandling struct {
	handling bool
	inline   bool
	done     bool
}

func main() {
	start := time.Now()

	// withParser(f)
	// withScanner(f)

	// s := NewScanner()

	// // // s.SetPackage("toto")

	// // f, err := os.Open("testdata/inline_imports.go")
	// f, err := os.Open("testdata/gomock.go")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// s.Scan(f)
	// f.Close()

	// f, err = os.Open("testdata/gomock.go")
	// // f, err = os.Open("testdata/inline_imports.go")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// s.Scan(f)
	// f.Close()

	s := NewScannerByteSlice()
	s.SetPackage("toto")

	b, err := os.ReadFile("testdata/gomock.go")
	if err != nil {
		log.Fatal(err)
	}
	s.ScanFile(b)

	b, err = os.ReadFile("testdata/inline_imports.go")
	if err != nil {
		log.Fatal(err)
	}
	s.ScanFile(b)

	// var buf strings.Builder
	// s.WriteTo(&buf)
	// fmt.Print(buf.String())

	f, err := os.Create("out.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	s.WriteTo(f)

	elapsed := time.Since(start)
	fmt.Printf("elapsed | %T | %v\n", elapsed, elapsed)
}

func withParser(w io.Reader) {
	fset := token.NewFileSet()
	toto, err := parser.ParseFile(fset, "", w, parser.ParseComments|parser.ImportsOnly)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("toto.Name | %T | %v\n", toto.Name, toto.Name)
	// fmt.Printf("toto.Doc | %T | %v\n", toto.Doc, toto.Doc)
	// fmt.Printf("toto.Comments | %T | %v\n", toto.Comments, toto.Comments)
	// for _, imp := range toto.Imports {
	// 	fmt.Printf("imp.Path.Value | %T | %v\n", imp.Path.Value, imp.Path.Value)
	// }
	// pos := toto.Imports[len(toto.Imports)-1].End()
	// fmt.Printf("pos | %T | %v\n", pos, pos)
}

func withScanner(w io.WriterTo) {
	var buf bytes.Buffer
	_, err := w.WriteTo(&buf)
	if err != nil {
		log.Fatal(err)
	}

	var pkg string
	var docbuf bytes.Buffer
	imprts := make(map[string]imprt, 15)
	var bodybuf bytes.Buffer

	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), buf.Len())
	var s scanner.Scanner
	s.Init(file, buf.Bytes(), nil, scanner.ScanComments)

	var imprtBloc imprtHandling
	var prev token.Token
	var imp imprt
	// Use wrote instead of docbuf.Len() because docbuf will grow larger with each file, cannot use total len as total wrote for file.
	var wrote int
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		if prev == token.PACKAGE && pkg == "" {
			pkg = lit
		}

		if tok == token.PACKAGE {
			if docbuf.Len() > 0 {
				docbuf.WriteString("\n")
			}
			wrote, _ = docbuf.Write(buf.Next(fset.Position(pos).Offset))
		}

		// If previous token was import & current is string or ident, means inline import.
		// Can handle
		// 	import "fmt"
		//
		// 	import "fmt"
		//	import "math/rand"
		//
		//	import m "math"
		//
		//	import f "fmt"
		//	import m "math"
		//
		//	import . "math"
		//
		//	import . "fmt"
		//	import . "math"
		//
		//	import _ "math"
		//
		//	import _ "fmt"
		//	import _ "math"
		if prev == token.IMPORT && (tok == token.STRING || tok == token.IDENT || tok == token.PERIOD) {
			imprtBloc.inline = true
		}

		// Start import saving.
		if tok == token.IMPORT {
			imprtBloc.handling = true
			imprtBloc.done = false
		}

		// If imports are done & we reached newline, write all remaining file to body buffer.
		if imprtBloc.done && prev == token.SEMICOLON {
			if bodybuf.Len() > 0 {
				buf.WriteString("\n")
			}
			// Scan does not read buffer, offset is from theorical start, not already consumed one. We manually skip all from start until current position. We did already read package documentation so we need to account for that when offseting.
			buf.Next(file.Offset(pos) - wrote)
			buf.WriteTo(&bodybuf)
			break
		}

		if imprtBloc.handling == true {
			if tok == token.IDENT {
				imp.Name = lit
			}

			if tok == token.PERIOD {
				imp.Name = "."
			}

			if tok == token.STRING {
				imp.Path = lit
				imprts[lit] = imp

				imp = imprt{}

				// If we were handling inline import, we can consider treatment of imports done.
				if imprtBloc.inline {
					imprtBloc.inline = false
					imprtBloc.handling = false
					imprtBloc.done = true
				}
			}

			// First losing parenthesis after import statement means we are done for parenthesis imports.
			if tok == token.RPAREN {
				imprtBloc.handling = false
				imprtBloc.done = true
			}
		}

		prev = tok
	}

	var bufout bytes.Buffer
	docbuf.WriteTo(&bufout)
	bufout.WriteString("package ")
	bufout.WriteString(pkg)
	bufout.WriteString("\n")
	bufout.WriteString("\n")
	// TODO imports
	if len(imprts) > 0 {
		bufout.WriteString("import (\n")
		for v := range maps.Values(imprts) {
			bufout.WriteString("\t")
			if v.Name != "" {
				bufout.WriteString(v.Name)
				bufout.WriteString(" ")
			}
			bufout.WriteString(v.Path)
			bufout.WriteString("\n")
		}
		bufout.WriteString(")\n")
		bufout.WriteString("\n")
	}
	bodybuf.WriteTo(&bufout)

	fmt.Println(bufout.String())
}

type Scanner interface {
	SetPackage(string)
	ScanFile(io.Reader)
	WriteTo(io.Writer) (int64, error)
}

type scnr struct {
	docbuf  bytes.Buffer
	pkgName string
	imports map[string]imprt
	bodybuf bytes.Buffer
}

func NewScanner() Scanner {
	return &scnr{
		imports: make(map[string]imprt, 15),
	}
}

func (sc *scnr) SetPackage(pkg string) {
	sc.pkgName = pkg
}

func (sc *scnr) ScanFile(r io.Reader) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		log.Fatal(err)
	}

	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), buf.Len())
	var s scanner.Scanner
	s.Init(file, buf.Bytes(), nil, scanner.ScanComments)

	var imprtBloc imprtHandling
	var prev token.Token
	var imp imprt
	// Use wrote instead of docbuf.Len() because docbuf will grow larger with each file, cannot use total len as total wrote for file.
	var wrote int
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		if prev == token.PACKAGE && sc.pkgName == "" {
			sc.pkgName = lit
		}

		if tok == token.PACKAGE {
			if sc.docbuf.Len() > 0 {
				sc.docbuf.WriteString("\n")
			}
			wrote, _ = sc.docbuf.Write(buf.Next(file.Position(pos).Offset))
		}

		// If previous token was import & current is string or ident, means inline import.
		// Can handle
		// 	import "fmt"
		//
		// 	import "fmt"
		//	import "math/rand"
		//
		//	import m "math"
		//
		//	import f "fmt"
		//	import m "math"
		//
		//	import . "math"
		//
		//	import . "fmt"
		//	import . "math"
		//
		//	import _ "math"
		//
		//	import _ "fmt"
		//	import _ "math"
		if prev == token.IMPORT && (tok == token.STRING || tok == token.IDENT || tok == token.PERIOD) {
			imprtBloc.inline = true
		}

		// Start import saving.
		if tok == token.IMPORT {
			imprtBloc.handling = true
			imprtBloc.done = false
		}

		// If imports are done & we reached newline, write all remaining file to body buffer.
		if imprtBloc.done && prev == token.SEMICOLON {
			if sc.bodybuf.Len() > 0 {
				sc.bodybuf.WriteString("\n")
			}
			// Scan does not read buffer, offset is from theorical start, not already consumed one. We manually skip all from start until current position. We did already read package documentation so we need to account for that when offseting.
			buf.Next(file.Offset(pos) - wrote)
			buf.WriteTo(&sc.bodybuf)
			break
		}

		if imprtBloc.handling == true {
			if tok == token.IDENT {
				imp.Name = lit
			}

			if tok == token.PERIOD {
				imp.Name = "."
			}

			if tok == token.STRING {
				imp.Path = lit
				sc.imports[lit] = imp

				imp = imprt{}

				// If we were handling inline import, we can consider treatment of imports done.
				if imprtBloc.inline {
					imprtBloc.inline = false
					imprtBloc.handling = false
					imprtBloc.done = true
				}
			}

			// First losing parenthesis after import statement means we are done for parenthesis imports.
			if tok == token.RPAREN {
				imprtBloc.handling = false
				imprtBloc.done = true
			}
		}

		prev = tok
	}
}

func (sc *scnr) WriteTo(w io.Writer) (int64, error) {
	var out int64
	i, err := sc.docbuf.WriteTo(w)
	if err != nil {
		return out, err
	}
	out += i

	tmp, err := io.WriteString(w, "package ")
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	tmp, err = io.WriteString(w, sc.pkgName)
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	tmp, err = io.WriteString(w, "\n\n")
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	if len(sc.imports) > 0 {
		tmp, err = io.WriteString(w, "import (\n")
		if err != nil {
			return out, err
		}
		out += int64(tmp)

		for v := range maps.Values(sc.imports) {
			tmp, err = io.WriteString(w, "\t")
			if err != nil {
				return out, err
			}
			out += int64(tmp)

			if v.Name != "" {
				tmp, err = io.WriteString(w, v.Name)
				if err != nil {
					return out, err
				}
				out += int64(tmp)
				tmp, err = io.WriteString(w, " ")
				if err != nil {
					return out, err
				}
				out += int64(tmp)
			}

			tmp, err = io.WriteString(w, v.Path)
			if err != nil {
				return out, err
			}
			out += int64(tmp)
			tmp, err = io.WriteString(w, "\n")
			if err != nil {
				return out, err
			}
			out += int64(tmp)
		}

		tmp, err = io.WriteString(w, ")\n\n")
		if err != nil {
			return out, err
		}
		out += int64(tmp)
	}

	i, err = sc.bodybuf.WriteTo(w)
	if err != nil {
		return out, err
	}
	out += i

	return out, nil
}

type ScannerByteSlice interface {
	SetPackage(string)
	ScanFile([]byte)
	WriteTo(io.Writer) (int64, error)
}

type scnrbs struct {
	docbuf  bytes.Buffer
	pkgName string
	imports map[string]imprt
	bodybuf bytes.Buffer
}

func NewScannerByteSlice() ScannerByteSlice {
	return &scnrbs{
		imports: make(map[string]imprt, 15),
	}
}

func (sc *scnrbs) SetPackage(pkg string) {
	sc.pkgName = pkg
}

func (sc *scnrbs) ScanFile(b []byte) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(b))
	var s scanner.Scanner
	s.Init(file, b, nil, scanner.ScanComments)

	var imprtBloc imprtHandling
	var prev token.Token
	var imp imprt
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		if prev == token.PACKAGE && sc.pkgName == "" {
			sc.pkgName = lit
		}

		if tok == token.PACKAGE {
			if sc.docbuf.Len() > 0 {
				sc.docbuf.WriteString("\n")
			}
			sc.docbuf.Write(b[:file.Position(pos).Offset])
		}

		// If previous token was import & current is string or ident, means inline import.
		// Can handle
		// 	import "fmt"
		//
		// 	import "fmt"
		//	import "math/rand"
		//
		//	import m "math"
		//
		//	import f "fmt"
		//	import m "math"
		//
		//	import . "math"
		//
		//	import . "fmt"
		//	import . "math"
		//
		//	import _ "math"
		//
		//	import _ "fmt"
		//	import _ "math"
		if prev == token.IMPORT && (tok == token.STRING || tok == token.IDENT || tok == token.PERIOD) {
			imprtBloc.inline = true
		}

		// Start import saving.
		if tok == token.IMPORT {
			imprtBloc.handling = true
			imprtBloc.done = false
		}

		// If imports are done & we reached newline, write all remaining file to body buffer.
		if imprtBloc.done && prev == token.SEMICOLON {
			if sc.bodybuf.Len() > 0 {
				sc.bodybuf.WriteString("\n")
			}
			sc.bodybuf.Write(b[file.Offset(pos):])
			break
		}

		if imprtBloc.handling == true {
			if tok == token.IDENT {
				imp.Name = lit
			}

			if tok == token.PERIOD {
				imp.Name = "."
			}

			if tok == token.STRING {
				imp.Path = lit
				sc.imports[lit] = imp

				imp = imprt{}

				// If we were handling inline import, we can consider treatment of imports done.
				if imprtBloc.inline {
					imprtBloc.inline = false
					imprtBloc.handling = false
					imprtBloc.done = true
				}
			}

			// First losing parenthesis after import statement means we are done for parenthesis imports.
			if tok == token.RPAREN {
				imprtBloc.handling = false
				imprtBloc.done = true
			}
		}

		prev = tok
	}
}

func (sc *scnrbs) WriteTo(w io.Writer) (int64, error) {
	var out int64
	i, err := sc.docbuf.WriteTo(w)
	if err != nil {
		return out, err
	}
	out += i

	tmp, err := io.WriteString(w, "package ")
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	tmp, err = io.WriteString(w, sc.pkgName)
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	tmp, err = io.WriteString(w, "\n\n")
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	if len(sc.imports) > 0 {
		tmp, err = io.WriteString(w, "import (\n")
		if err != nil {
			return out, err
		}
		out += int64(tmp)

		for v := range maps.Values(sc.imports) {
			tmp, err = io.WriteString(w, "\t")
			if err != nil {
				return out, err
			}
			out += int64(tmp)

			if v.Name != "" {
				tmp, err = io.WriteString(w, v.Name)
				if err != nil {
					return out, err
				}
				out += int64(tmp)
				tmp, err = io.WriteString(w, " ")
				if err != nil {
					return out, err
				}
				out += int64(tmp)
			}

			tmp, err = io.WriteString(w, v.Path)
			if err != nil {
				return out, err
			}
			out += int64(tmp)
			tmp, err = io.WriteString(w, "\n")
			if err != nil {
				return out, err
			}
			out += int64(tmp)
		}

		tmp, err = io.WriteString(w, ")\n\n")
		if err != nil {
			return out, err
		}
		out += int64(tmp)
	}

	i, err = sc.bodybuf.WriteTo(w)
	if err != nil {
		return out, err
	}
	out += i

	return out, nil
}
