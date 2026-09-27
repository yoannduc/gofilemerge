package merger

import (
	"bytes"
	"go/scanner"
	"go/token"
	"io"
	"maps"
	"slices"
	"strings"
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

type Merger interface {
	SetPackage(string)
	Write([]byte) (int, error)
	WriteTo(io.Writer) (int64, error)
}

type merger struct {
	docbuf  bytes.Buffer
	pkgName string
	imports map[string]imprt
	bodybuf bytes.Buffer
}

var _ Merger = (*merger)(nil)

func NewMerger() Merger {
	return &merger{
		imports: make(map[string]imprt, 15),
	}
}

func (m *merger) SetPackage(pkg string) {
	m.pkgName = pkg
}

func (m *merger) Write(p []byte) (int, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(p))
	var s scanner.Scanner
	s.Init(file, p, nil, scanner.ScanComments)

	var n int

	var imprtBloc imprtHandling
	var prev token.Token
	var imp imprt
	var skipFrom int
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		if prev == token.PACKAGE && m.pkgName == "" {
			m.pkgName = lit
		}

		if tok == token.PACKAGE {
			if file.Offset(pos) > 0 && m.docbuf.Len() > 0 {
				_, err := m.docbuf.WriteString("\n")
				if err != nil {
					return n, err
				}
			}
			i, err := m.docbuf.Write(p[:file.Offset(pos)])
			if err != nil {
				return n, err
			}
			n += i
			skipFrom = file.Offset(pos)
		}

		// If previous token was import & current is not `(`, means inline import.
		// Can handle any form of
		//
		// 	import "math"
		//
		//	import m "math"
		//
		//	import . "math"
		//
		//	import _ "math"
		//
		// Also handles multi line inline imports like
		//
		//	import "math"
		//	import "fmt"
		//
		if prev == token.IMPORT && tok != token.LPAREN {
			imprtBloc.inline = true
		}

		// Start import saving.
		if tok == token.IMPORT {
			imprtBloc.handling = true
			imprtBloc.done = false
		}

		// If imports are done & we reached newline, write all remaining file to body buffer.
		if imprtBloc.done && prev == token.SEMICOLON {
			if m.bodybuf.Len() > 0 {
				_, err := m.bodybuf.WriteString("\n")
				if err != nil {
					return n, err
				}
			}
			i, err := m.bodybuf.Write(p[file.Offset(pos):])
			if err != nil {
				return n, err
			}
			n += i
			n += len(p[skipFrom:file.Offset(pos)])
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
				m.imports[lit] = imp

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

	return n, nil
}

func (m *merger) WriteTo(w io.Writer) (int64, error) {
	var out int64
	i, err := m.docbuf.WriteTo(w)
	if err != nil {
		return out, err
	}
	out += i

	tmp, err := io.WriteString(w, "package ")
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	tmp, err = io.WriteString(w, m.pkgName)
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	tmp, err = io.WriteString(w, "\n\n")
	if err != nil {
		return out, err
	}
	out += int64(tmp)

	if len(m.imports) > 0 {
		tmp, err = io.WriteString(w, "import (\n")
		if err != nil {
			return out, err
		}
		out += int64(tmp)

		// Sort to have repeatable output for tests
		for _, v := range slices.SortedFunc(maps.Values(m.imports), func(a, b imprt) int {
			return strings.Compare(a.Path, b.Path)
		}) {
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

	i, err = m.bodybuf.WriteTo(w)
	if err != nil {
		return out, err
	}
	out += i

	return out, nil
}
