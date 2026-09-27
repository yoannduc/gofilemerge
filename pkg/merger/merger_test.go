package merger

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestNewMerger(t *testing.T) {
	v := NewMerger()
	r := reflect.ValueOf(v)
	if _, ok := reflect.TypeAssert[Merger](r); !ok {
		t.Fatalf(`err1`)
	}
	if _, ok := reflect.TypeAssert[*merger](r); !ok {
		t.Fatalf(`err2`)
	}

	new := &merger{
		imports: make(map[string]imprt, 15),
	}
	if !reflect.DeepEqual(v, new) {
		t.Fatalf(`err3`)
	}
}

func TestSetPackage(t *testing.T) {
	tc := []struct {
		in  string
		out string
	}{
		{"lower", "lower"},
		{"UPPER", "UPPER"},
		{"MiXeD", "MiXeD"},
		{"Spécial_%!@🙃", "Spécial_%!@🙃"},
	}

	for _, test := range tc {
		t.Run(test.in, func(t *testing.T) {
			m := NewMerger()
			m.SetPackage(test.in)
			cast, ok := m.(*merger)
			if !ok {
				t.Fatalf("could not cast Merger to *merger")
			}
			if !reflect.DeepEqual(cast.pkgName, test.out) {
				t.Fatalf(`value was not expected for input "%v". Expected %v, got %v`, test.in, test.out, cast.pkgName)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	type inout struct {
		in  []byte
		out int
		err error
	}

	tc := map[string]struct {
		inout   []inout
		docbuf  bytes.Buffer
		pkgName string
		imports map[string]imprt
		bodybuf bytes.Buffer
	}{
		"empty": {
			[]inout{
				{
					[]byte(``),
					0,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			"",
			map[string]imprt{},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
		},
		"[one] package only uppercase": {
			[]inout{
				{
					[]byte(`package UPPER`),
					0,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			"UPPER",
			map[string]imprt{},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
		},
		"[one] package with one line import": {
			[]inout{
				{
					[]byte(`package pkg
import "math"`),
					0,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			"pkg",
			map[string]imprt{
				`"math"`: {
					"",
					`"math"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
		},
		"[one] package with multiple line import": {
			[]inout{
				{
					[]byte(`package pkg
import _ "math"
import . "custom/path/package"`),
					0,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			"pkg",
			map[string]imprt{
				`"math"`: {
					"_",
					`"math"`,
				},
				`"custom/path/package"`: {
					".",
					`"custom/path/package"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
		},
		"[one] package with imports": {
			[]inout{
				{
					[]byte(`package pkg
import (
"math"
"fmt"
)`),
					0,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			"pkg",
			map[string]imprt{
				`"math"`: {
					"",
					`"math"`,
				},
				`"fmt"`: {
					"",
					`"fmt"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
		},
		"[one] pkg doc with package with imports": {
			[]inout{
				{
					[]byte(`//string doc
package pkg
import (
"math"
"fmt"
)`),
					13,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString("//string doc\n")
				return tmp
			}(),
			"pkg",
			map[string]imprt{
				`"math"`: {
					"",
					`"math"`,
				},
				`"fmt"`: {
					"",
					`"fmt"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
		},
		"[one] pkg doc with package with imports & body": {
			[]inout{
				{
					[]byte(`//string doc
package pkg
import (
"math"
"fmt"
)
body is not necessary valid go`),
					79,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString("//string doc\n")
				return tmp
			}(),
			"pkg",
			map[string]imprt{
				`"math"`: {
					"",
					`"math"`,
				},
				`"fmt"`: {
					"",
					`"fmt"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString("body is not necessary valid go")
				return tmp
			}(),
		},
		"[multiple] 3 small files": {
			[]inout{
				{
					[]byte(`// multi
// line
// string
// doc
package pkg1
import (
"fmt"
)

func main() {
fmt.Println("Hello World !")
}`),
					109,
					nil,
				},
				{
					[]byte(`package pkg2
import (
"math"
"fmt"
)

func main() {
fmt.Println(math.Pi)
}
`),
					75,
					nil,
				},
				{
					[]byte(`// more doc
package pkg3
import (
"log"
"maps"
"math"
)

func main() {
for v := range maps.Values(map[string]float64{"": 38.5}) {
log.Println(math.Round(v))
}
}
`),
					161,
					nil,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString(`// multi
// line
// string
// doc

// more doc
`)
				return tmp
			}(),
			"pkg1",
			map[string]imprt{
				`"fmt"`: {
					"",
					`"fmt"`,
				},
				`"log"`: {
					"",
					`"log"`,
				},
				`"math"`: {
					"",
					`"math"`,
				},
				`"maps"`: {
					"",
					`"maps"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString(`func main() {
fmt.Println("Hello World !")
}
func main() {
fmt.Println(math.Pi)
}

func main() {
for v := range maps.Values(map[string]float64{"": 38.5}) {
log.Println(math.Round(v))
}
}
`)
				return tmp
			}(),
		},
	}

	for name, test := range tc {
		t.Run(name, func(t *testing.T) {
			m := NewMerger()
			for _, tst := range test.inout {
				v, err := m.Write(tst.in)
				if !reflect.DeepEqual(v, tst.out) {
					t.Fatalf(`wrong number of bytes written. Expected %v, got %v`, tst.out, v)
				}
				if err != nil && !errors.Is(err, tst.err) {
					t.Fatalf(`unexpected error. Expected "%v", got "%v"`, tst.err, err)
				}
			}

			cast, ok := m.(*merger)
			if !ok {
				t.Fatalf("could not cast Merger to *merger")
			}

			if !reflect.DeepEqual(cast.docbuf, test.docbuf) {
				t.Fatalf(`docubuf value error. Expected %v, got %v`, test.docbuf, cast.docbuf)
			}
			if !reflect.DeepEqual(cast.pkgName, test.pkgName) {
				t.Fatalf(`pkgName value error. Expected %v, got %v`, test.pkgName, cast.pkgName)
			}
			if !reflect.DeepEqual(cast.imports, test.imports) {
				t.Fatalf(`imports value error. Expected %v, got %v`, test.imports, cast.imports)
			}
			if !reflect.DeepEqual(cast.bodybuf, test.bodybuf) {
				t.Fatalf(`bodybuf value error. Expected %v, got %v`, test.bodybuf, cast.bodybuf)
			}
		})
	}
}

func TestWriteTo(t *testing.T) {
	type inout struct {
		in  []byte
		out int
		err error
	}

	tc := map[string]struct {
		docbuf   bytes.Buffer
		pkgName  string
		imports  map[string]imprt
		bodybuf  bytes.Buffer
		out      int64
		err      error
		expected string
	}{
		"empty (with package name cause cannot test if empty)": {
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			// cannot test if empty, causes to write `package `
			// but cannot use trailing space in go string for test
			"name",
			map[string]imprt{},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			14,
			nil,
			`package name

`,
		},
		"with docstring": {
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString(`// this
// is
// doc
// string
`)
				return tmp
			}(),
			// cannot test if empty, causes to write `package `
			// but cannot use trailing space in go string for test
			"name",
			map[string]imprt{},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			45,
			nil,
			`// this
// is
// doc
// string
package name

`,
		},
		"with body": {
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			// cannot test if empty, causes to write `package `
			// but cannot use trailing space in go string for test
			"name",
			map[string]imprt{},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString(`random body
`)
				return tmp
			}(),
			26,
			nil,
			`package name

random body
`,
		},
		"with imports": {
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			// cannot test if empty, causes to write `package `
			// but cannot use trailing space in go string for test
			"name",
			map[string]imprt{
				`"fmt"`: {
					".",
					`"fmt"`,
				},
				`"math"`: {
					"_",
					`"math"`,
				},
				`"maps"`: {
					"",
					`"maps"`,
				},
				`"path/to/package"`: {
					"pkg",
					`"path/to/package"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				return tmp
			}(),
			76,
			nil,
			`package name

import (
	. "fmt"
	"maps"
	_ "math"
	pkg "path/to/package"
)

`,
		},
		"a bit of everything": {
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString(`// doc1

// doc2
// doc2

// doc3
`)
				return tmp
			}(),
			// cannot test if empty, causes to write `package `
			// but cannot use trailing space in go string for test
			"name",
			map[string]imprt{
				`"fmt"`: {
					".",
					`"fmt"`,
				},
				`"math"`: {
					"_",
					`"math"`,
				},
				`"maps"`: {
					"",
					`"maps"`,
				},
				`"path/to/package"`: {
					"pkg",
					`"path/to/package"`,
				},
			},
			func() bytes.Buffer {
				var tmp bytes.Buffer
				tmp.WriteString(`func main() {
fmt.Println("Hello World !")
}

func main() {
fmt.Println(math.Pi)
}

func main() {
for v := range maps.Values(map[string]float64{"": 38.5}) {
log.Println(math.Round(v))
}
}
`)
				return tmp
			}(),
			298,
			nil,
			`// doc1

// doc2
// doc2

// doc3
package name

import (
	. "fmt"
	"maps"
	_ "math"
	pkg "path/to/package"
)

func main() {
fmt.Println("Hello World !")
}

func main() {
fmt.Println(math.Pi)
}

func main() {
for v := range maps.Values(map[string]float64{"": 38.5}) {
log.Println(math.Round(v))
}
}
`,
		},
	}

	for name, test := range tc {
		t.Run(name, func(t *testing.T) {
			m := NewMerger()

			cast, ok := m.(*merger)
			if !ok {
				t.Fatalf("could not cast Merger to *merger")
			}
			cast.docbuf = test.docbuf
			cast.pkgName = test.pkgName
			cast.imports = test.imports
			cast.bodybuf = test.bodybuf

			var buf strings.Builder
			v, err := m.WriteTo(&buf)
			if !reflect.DeepEqual(v, test.out) {
				t.Fatalf(`wrong number of bytes written. Expected %v, got %v`, test.out, v)
			}
			if err != nil && !errors.Is(err, test.err) {
				t.Fatalf(`unexpected error. Expected "%v", got "%v"`, test.err, err)
			}

			out := buf.String()
			if !reflect.DeepEqual(out, test.expected) {
				fmt.Printf("out \"%v\"\n", out)
				fmt.Printf("expected\"%v\"\n", test.expected)
				t.Fatalf(`wrong number of bytes written. Expected %v, got %v`, test.expected, out)
			}
		})
	}
}
