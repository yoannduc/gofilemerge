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

// imprt is a struct that represent one import, with Name being its custom
// name & Path being its complete path, like:
//
//	import (
//		gin /* <- Name */ "github.com/gin-gonic/gin" /* <- Path */
//	)
type imprt struct {
	Name string
	Path string
}

// imprtHandling is a struct to keep track of state of import being
// handled or not.
type imprtHandling struct {
	handling bool
	inline   bool
	done     bool
}

// A Merger is used to merge files. Files to be merged are to be added
// to buffer through Write method, then when all files are added, the
// output merged file can be written to an [io.Writer] with WriteTo.
// Merger is not thread safe.
//
// For imports, it handles both group imports & inline ones. It also handles multi-lines imports like:
//
//	import "fmt"
//	import "math"
//
// It handles aliased, dot and blank imports like:
//
//	import "math"
//	import m "math"
//	import . "math"
//	import _ "math"
type Merger interface {
	SetPackage(string)
	Write([]byte) (int, error)
	WriteTo(io.Writer) (int64, error)
}

// merger is the concrete type that implements [Merger].
type merger struct {
	docbuf  bytes.Buffer
	pkgName string
	imports map[string]imprt
	bodybuf bytes.Buffer
}

// Check that *merger correctly implements [Merger].
var _ Merger = (*merger)(nil)

// NewMerger creates and initializes a new [Merger].
func NewMerger() Merger {
	return &merger{
		imports: make(map[string]imprt, 15),
	}
}

// SetPackage sets the package name that will appear on merged output.
func (m *merger) SetPackage(pkg string) {
	m.pkgName = pkg
}

// Write implements [io.Writer].
//
// Write writes a go file to [Merger]. The return value n is the
// length of p; err is always nil.
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
				_, _ = m.docbuf.WriteString("\n")
			}
			i, _ := m.docbuf.Write(p[:file.Offset(pos)])
			n += i
			skipFrom = file.Offset(pos)
		}

		// If previous token was import & current is not `(`, means
		// inline import.
		if prev == token.IMPORT && tok != token.LPAREN {
			imprtBloc.inline = true
		}

		// Start import saving.
		if tok == token.IMPORT {
			imprtBloc.handling = true
			imprtBloc.done = false
		}

		// If imports are done & we reached newline, write all
		// remaining file to body buffer.
		if imprtBloc.done && prev == token.SEMICOLON {
			if m.bodybuf.Len() > 0 {
				_, _ = m.bodybuf.WriteString("\n")
			}
			i, _ := m.bodybuf.Write(p[file.Offset(pos):])
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

				// If we were handling inline import, we can consider
				// treatment of imports done.
				if imprtBloc.inline {
					imprtBloc.inline = false
					imprtBloc.handling = false
					imprtBloc.done = true
				}
			}

			// First losing parenthesis after import statement means we
			// are done for parenthesis imports.
			if tok == token.RPAREN {
				imprtBloc.handling = false
				imprtBloc.done = true
			}
		}

		prev = tok
	}

	return n, nil
}

// WriteTo implements [io.WriterTo].
//
// WriteTo writes resulting merged file to w until there's no more
// data to write or when an error occurs. The return value n is the
// number of bytes written. Any error encountered during the write
// is also returned.
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
