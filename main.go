package main

import (
	"compress/zlib"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
)

func main() {
	cmds := map[string]func([]string){
		"decompress": decompressCmd,
		"cat-file":   catFile,
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

	reader, err := zlib.NewReader(f)
	if err != nil {
		logToStdErrAndExit("error decompressing %s: %v", p, err)
		return
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)
	_ = pretty
}

func logToStdErrAndExit(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
