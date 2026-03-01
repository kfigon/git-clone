package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
)

func main() {
	cmds := map[string]func([]string){
		"decompress":  decompressCmd,
		"cat-file":    catFile,
		"hash-object": hashObject,
		"ls-tree":     lsTree,
	}

	if len(os.Args) < 2 {
		logToStdErrAndExit("command not provided, available: %s", availableCmds(cmds))
		return
	}

	if fn, ok := cmds[os.Args[1]]; !ok {
		logToStdErrAndExit("invalid command %s, available: %s", os.Args[1], availableCmds(cmds))
		return
	} else {
		fn(os.Args[2:])
	}
}

func availableCmds(cmds map[string]func([]string)) string {
	return strings.Join(slices.Collect(maps.Keys(cmds)), ", ")
}

func decompressCmd(_ []string) {
	reader, err := zlib.NewReader(os.Stdin)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)
}

func catFile(s []string) {
	// blobs, trees, commits - types of objects
	fs := flag.NewFlagSet("cat-file", flag.ExitOnError)
	pretty := fs.Bool("p", false, "pretty print object")
	fs.Parse(s)

	restOfArgs := fs.Args()
	if len(restOfArgs) < 1 {
		logToStdErrAndExit("sha-1 of object not provided")
		return
	}
	sha := fs.Arg(0)
	if len(sha) < 3 {
		logToStdErrAndExit("invalid sha-1 provided")
		return
	}

	// todo: support shortest unique sha, do not require a full hash

	// git optimises for linux file structure. Linux does not like a lot of files in dir, so first 2 chars of the char is dir, then rest as filenam
	p := path.Join(".git", "objects", sha[:2], sha[2:])
	f, err := os.Open(p)
	if err != nil {
		logToStdErrAndExit("error reading %s: %v", p, err)
		return
	}
	defer f.Close()

	decompressed, err := zlib.NewReader(f)
	if err != nil {
		logToStdErrAndExit("error decompressing %s: %v", p, err)
		return
	}
	defer decompressed.Close()

	reader := bufio.NewReader(decompressed)
	// blob format - there's a cstring:
	// blob <size>0<content>
	header, err := reader.ReadBytes(0)
	if err != nil {
		logToStdErrAndExit("error reading blob header %s: %v", p, err)
		return
	}

	// skip 0 byte
	header = header[:len(header)-1]
	headerParts := strings.Split(string(header), " ")
	if len(headerParts) != 2 {
		logToStdErrAndExit("invalid format of header: %s", header)
		return
	}

	kind := headerParts[0]
	size, err := strconv.Atoi(headerParts[1])
	if err != nil {
		logToStdErrAndExit("error parsing size of blob %s: %v", p, err)
		return
	}

	switch kind {
	case "blob":
		_ = pretty // todo: this is already a pretty print, will do different when supporting modes

		// limit reader to avoid allocations and zlib bomb
		limited := io.LimitReader(reader, int64(size))
		io.Copy(os.Stdout, limited)
	default:
		logToStdErrAndExit("unknown type %q for hash %s", kind, p)
		return
	}
}

func logToStdErrAndExit(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func hashObject(args []string) {
	fs := flag.NewFlagSet("hash-object", flag.ExitOnError)
	write := fs.Bool("w", false, "set to true to write to file")
	fs.Parse(args)

	fileName := fs.Arg(0)
	if len(fileName) == 0 {
		logToStdErrAndExit("no file provided")
		return
	}

	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		logToStdErrAndExit("error reading file %s: %v", fileName, err)
		return
	}
	// blob <size>0<content>
	// hash it to get the key (20 bytes)
	// compress the it and store in the file if -w provided

	size := len(fileBytes)

	buf := bytes.NewBuffer(make([]byte, 0, size+len("blob ")+1))
	buf.WriteString(fmt.Sprintf("blob %d", size))
	buf.WriteByte(0)
	buf.Write(fileBytes)
	finalHash := sha1.Sum(buf.Bytes())
	printableHash := hex.EncodeToString(finalHash[:])

	if *write {
		compressed := bytes.NewBuffer(nil)
		compressor := zlib.NewWriter(compressed)

		compressor.Write([]byte(fmt.Sprintf("blob %d", size)))
		compressor.Write([]byte{0})
		compressor.Write(fileBytes)
		compressor.Close()

		dir := path.Join(".git", "objects", printableHash[:2])
		// this won't error if file is already created
		if err = os.MkdirAll(dir, 0755); err != nil {
			logToStdErrAndExit("error creating directory %s: %w", dir, err)
			return
		}

		p := path.Join(dir, printableHash[2:])
		f, err := os.Create(p)
		if err != nil {
			logToStdErrAndExit("error creating file %s: %v", p, err)
			return
		}
		defer f.Close()

		if _, err = io.Copy(f, compressed); err != nil {
			logToStdErrAndExit("error writing contet to file", p, err)
			return
		}
	}

	fmt.Println(printableHash)
}

func lsTree(args []string) {
	// todo: https://youtu.be/u0VotuGzD_w?t=6408
}
