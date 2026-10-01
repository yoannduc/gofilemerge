# gofilemerge

[![LICENSE](https://img.shields.io/badge/License-MIT-turquise.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/yoannduc/gofilemerge/pkg/merger.svg)](https://pkg.go.dev/github.com/yoannduc/gofilemerge/pkg/merger)

gofilemerge is a CLI tool built to merge multiple Go files together. This project originates from the need to merge multiple mocks mocking different packages' interfaces into one single file not to pollute the source tree.

## Installation

Install the `gofilemerge` tool:

```
go install github.com/yoannduc/gofilemerge@latest
```

Please note that gofilemerge uses iterators released in version [1.23](https://go.dev/doc/go1.23#iterators).

## Usage

```
gofilemerge [flags] [path ...]
  -out string
    	output file; defaults to stdout
  -pkg string
    	output package name; defaults to first package name scanned
  -rm
    	remove merged files
```

Example:

```
gofilemerge -pkg=custom -out=merged.go file1.go file2.go
```

Or in code with go generate:

```go
package pkg

//go:generate gofilemerge -pkg=mock_pkg -out=mock_pkg.go -rm mock_repo.go mock_mapper.go

type service struct {
	repo RepoInterface
	mapper MapperInterface
}
```
