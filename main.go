package main

import (
	"compress/zlib"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
)

func main() {
	cmds := map[string]func([]string){
		"decompress": decompressCmd,
		"cat-file":   catFile,
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "command not provided, available: %s\n",
			strings.Join(slices.Collect(maps.Keys(cmds)), ", "),
		)
		os.Exit(1)
		return
	}

	if fn, ok := cmds[os.Args[1]]; !ok {
		fmt.Fprintf(os.Stderr, "invalid command %s, available: %s\n",
			os.Args[1],
			strings.Join(slices.Collect(maps.Keys(cmds)), ", "),
		)
		os.Exit(1)
		return
	} else {
		fn(os.Args[2:])
	}
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

}
