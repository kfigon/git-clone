package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
)

type File struct {
	data []byte
}

func NewFile() *File {
	return &File{data: []byte{}}
}
func (f *File) Write(b []byte) {
	f.data = b
}
func (f *File) Read() []byte {
	return f.data
}

func (f *File) WriteS(s string) {
	f.data = []byte(s)
}
func (f *File) ReadS() string {
	return string(f.data)
}

type FileName string
type Directory map[FileName]*File

type Git struct {
	Objects Directory
	Refs    Directory
	Head    *File
}

func hashStr(s string) string {
	return hashBytes([]byte(s))
}

func hashBytes(b []byte) string {
	return fmt.Sprintf("%x", sha1.New().Sum(b))[:40]
}

type ObjectType int

const (
	Blob ObjectType = iota
	Tree
	Commit
)

func NewGit() *Git {
	return &Git{
		// all of these dirs and files will be under .git dir
		// linux optimisation in git: each directory has subdirectory
		// with 2 characters for each file
		//  so object abc123456 will be stored in .git/objects/ab/c123456
		// this is because linux does not like a lot of files in single dir
		// we'll skip that, store hash as file directly
		Objects: Directory{},
		Refs:    Directory{},
		Head:    NewFile(),
	}
}

func (g *Git) Init() {
	// create directories
	g.Head.WriteS("ref: refs/heads/master\n")
}

// read object
func (g *Git) CatFile(kind ObjectType, hash string) (string, error) {
	switch kind {
	case Blob:
		f := g.Objects[FileName(hash)]
		reader, err := zlib.NewReader(bytes.NewBuffer(f.Read()))
		if err != nil {
			return "", fmt.Errorf("failed to decompress file %s: %w", hash, err)
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("error reading file %s: %w", hash, err)
		}
		return string(data), nil
	}
	return "", nil
}

func (g *Git) WriteBlob(data []byte) error {
	f, ok := g.Objects[FileName(hashBytes(data))]
	if !ok {
		f = NewFile()
	}

	buf := bytes.NewBuffer(nil)
	writer := zlib.NewWriter(buf)
	writer.Close()
	writer.Write(data)

	b := &strings.Builder{}

	// blob <size>null<content>. Header is just a c-string
	// size is decimal encoded string
	b.WriteString(fmt.Sprintf("blob %d\x00", len(data)))
	// f.Write()
	_ = f
	return nil
}
