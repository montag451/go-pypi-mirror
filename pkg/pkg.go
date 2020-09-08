package pkg

import (
	"fmt"
	"path/filepath"
)

type Pkg struct {
	Path     string
	Filename string
	Metadata *Metadata
}

func New(path string) (*Pkg, error) {
	meta, err := getMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("error while processing %q: %w", path, err)
	}
	return &Pkg{path, filepath.Base(path), meta}, nil
}
