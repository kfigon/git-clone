package main

import (
	"bufio"
	"compress/zlib"
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
	headerParts := strings.Split(string(header[:len(header)-1]), " ")
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

	if kind == "blob" {
		_ = pretty // todo: this is already a pretty print, will do different when supporting modes

		// limit reader to avoid allocations
		limited := io.LimitReader(reader, int64(size))
		io.Copy(os.Stdout, limited)
	} else {
		logToStdErrAndExit("unknown type %q for hash %s", kind, p)
		return
	}

}

func logToStdErrAndExit(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func hashObject(args []string) {
	//todo: https://youtu.be/u0VotuGzD_w?t=4128
}
