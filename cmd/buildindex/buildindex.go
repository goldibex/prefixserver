package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	index "github.com/goldibex/prefixserver/index"
	"os"
	"path"
	"strings"
	"time"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-q] < index_source > binary_index\n\n", path.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "this program reads from stdin and writes to stdout.\n")
		fmt.Fprintf(os.Stderr, "it expects its input to be a variable-length newline-separated text file in the following format:\n")
		fmt.Fprintf(os.Stderr, "\n<variable name> <score>\n\n")
		fmt.Fprintf(os.Stderr, "where <variable name> is a Java variable name and <score> is an integer.\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")

		flag.PrintDefaults()
	}
}

func main() {

	quiet := flag.Bool("q", false, "Suppress non-fatal messages")
	startTime := time.Now()

	flag.Parse()
	stat, _ := os.Stdout.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintf(os.Stderr, "%s: won't write index to a terminal\n", path.Base(os.Args[0]))
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	nextWord := ""
	nextScore := 0

	in := index.New()

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Now building index. Each . represents 10,000 entries.\n")
	}
	entriesAdded := 0

	for scanner.Scan() {
		entriesAdded++
		if entriesAdded%10000 == 0 && !*quiet {
			fmt.Fprintf(os.Stderr, ".")
		}
		if count, err := fmt.Sscanf(scanner.Text(), "%s %d", &nextWord, &nextScore); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input: ", err)
			os.Exit(1)
		} else if count != 2 {
			fmt.Fprintf(os.Stderr, "Invalid line: '%s'\n", scanner.Text())
			os.Exit(1)
		} else {
			wordAsBytes := []byte(nextWord)
			in.Add(wordAsBytes, wordAsBytes, nextScore)

			// also add the underscored prefixes
			part := nextWord

			for {
				nextPrefixPos := strings.Index(part, "_")
				if nextPrefixPos == -1 {
					break
				}
				part = part[nextPrefixPos+1:]
				in.Add([]byte(part), wordAsBytes, nextScore)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "reading standard input: \n", err)
		os.Exit(1)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "\nCompacting index...\n")
	}
	in.Compact()

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Encoding index...\n")
	}

	enc := gob.NewEncoder(os.Stdout)
	if err := enc.Encode(in); err != nil {
		fmt.Fprintln(os.Stderr, "gob encoding index: ", err)
		os.Exit(1)
	}
	if !*quiet {
		fmt.Fprintf(os.Stderr, "Finished in %2f s\n", time.Since(startTime).Seconds())
	}

}
