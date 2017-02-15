package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	index "github.com/goldibex/alation/prefixserver/index"
	"os"
	"path"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s index_name\n\n", path.Base(os.Args[0]))
	}
}

func main() {

	flag.Parse()

	if flag.Arg(0) == "" {
		flag.Usage()
		os.Exit(2)
	}

	in := index.Index{}
	reader, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %s\n", flag.Arg(0), err)
		os.Exit(1)
	}
	dec := gob.NewDecoder(reader)
	if err = dec.Decode(&in); err != nil {
		fmt.Fprintf(os.Stderr, "opening %s: %s\n", flag.Arg(0), err)
		os.Exit(1)
	}

	inReader := bufio.NewReader(os.Stdin)

	values := make([][]byte, 10)
	scores := make([]int, 10)

	for {
		fmt.Print("> ")
		text, _ := inReader.ReadString('\n')
		if len(text) == 0 {
			fmt.Println("err: no query")
			continue
		}
		count := in.Find([]byte(text[:len(text)-1]), values, scores)
		if count == 0 {
			fmt.Printf("err: no matches for '%s'\n", []byte(text))
		} else {
			for i := 0; i < count; i++ {
				fmt.Printf("%s %d\n", values[i], scores[i])
			}
		}
	}

}
