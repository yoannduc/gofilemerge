package merger

import (
	"bytes"
	"go/scanner"
	"go/token"
	"io"
	"maps"
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
