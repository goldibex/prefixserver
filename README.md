prefixserver
============

My implementation of part 1 of Alation's code challenge.

## Dependencies

Just [Go](https://golang.org/doc/install) and its standard library. No
Gemfile or left-pad required.

## Features

- A REST/JSON HTTP server to handle lookups.
- TLS and HTTP/2 support.
- CPU and memory profiling built-in (that's a [Go freebie](https://golang.org/pkg/net/http/pprof) ).
- Gracefully scales to thousands of concurrent lookups per node per second,
if you have the network and CPU for it. (That's mostly Go, not me.)
- At a few million entries, still only requires about a millisecond per lookup on my anemic Macbook.
At that size the index is memory-hungry though.
- Per-request logging.

## Installation

```bash
$ go install github.com/goldibex/prefixserver
$ go install github.com/goldibex/prefixserver/cmd/buildindex
$ go install github.com/goldibex/prefixserver/cmd/checkindex
```

This will fetch the sources for prefixserver and build three binaries:

- `prefixserver`, the REST/JSON web server for the index
- `buildindex`, a tool for building a binary index from source files
- `checkindex`, a tool to query the index file directly.

## Usage

Before firing up the server, we'll need to build the index. The expected format
for index sources is a newline-delimited text file with lines as follows:
`<variable_name> <score>`

Included in this repository is an awful Perl script (is there any other kind?) for making a fake index source
with a couple million entries. You can use it as follows:

```bash
$ cd $GOPATH/src/github.com/goldibex/prefixserver
$ perl generate_fake_index.pl > sample.indexsource
```

The variables it produces look right at home in Perl.

Anyway, you can then generate an index binary from the source file:

```bash
$ buildindex < index_source > output.index
```

And use it in the HTTP server:
```
$ prefixserver output.index
```

## Deployment and management

Go's embedded HTTP server is pretty dynamite, so in the case of this app there's no need to reverse-proxy
for performance. That said, if you still need to put prefixserver behind a proxy for other reasons (complex
SSL termination, rollup with other HTTP servers, etc.), Docker makes things pretty simple. I've
included a Dockerfile in this repository that will build the app, no Go required. So you can do:

```bash
$ docker build -t prefixserver .
```

and the binaries will all be accessible via `docker run`.

Any deployment will have to consider how the servers access the index itself.
To expose a single file to a Docker container from the host is pretty straightforward (the -v flag),
but where does the file come from to begin with? In production, we could pull it from a secure network source
before starting the server. Doing this securely is highly environment-dependent. In cloud environments like
AWS you could use machine-based IAM authentication and pull from a locked-down S3 bucket that only your index-
_building_ system has write access to.

## Testing and profiling

There's a decent (not perfect) battery of units and benchmarks for the index. Run them with:

```bash
$ go test github.com/goldibex/alation/prefixserver/index -bench .
```
 
CI for this project is provided by CircleCI.

If you launch the server using the `-profile` flag, you'll be able to access CPU, heap, goroutine, and thread blocking
profiles via a separate HTTP server running at localhost:6060.

## Implementation notes

The index I implemented is a pretty run-of-the-mill unbalanced Patricia trie with value leaves.
To achieve total ordering of the output
over item scores, every non-leaf node is tagged with the maximum score of its descendants; then I use a priority
queue ("heap") for best-first traversal of the tree. As a result no sort is needed at the end, but the initial tree traversal
is costlier than it would be using a regular queue. TANSTAAFL.

Its search time in the number of elements in the index is definitely sublinear but not that easy to analyze, at least
not for an ancient Greek major.
Additional papers on the subject containing some _very_ pretty squiggles are available [here](https://arxiv.org/pdf/1303.4244.pdf) and [here](http://docs.lib.purdue.edu/cgi/viewcontent.cgi?article=1619&context=cstech).

## Future development

Nothing in prefixserver validates that the variable names it's given are, in fact, valid Java identifiers.
I peeked in the Java Language Specification to see how hard it would be to implement that validation in Go.
Identifier validity [is literally defined by the behavior of two library methods, `java.lang.Character.isJavaIdentifierStart(int)` and `java.lang.Character.isJavaIdentifierPart(int)`](https://docs.oracle.com/javase/specs/jls/se8/html/jls-3.html#jls-3.8).
Finding the actual sources for these methods in OpenJDK is known to cause reality inversions, and the errata
list 63 cases in which accidentally understanding all the bitwise voodoo
can cause an XK-class end-of-the-world scenario.

But you don't have to take my word for it! Go read the implementations of Character, CharacterData, CharacterDataLatin1,
and friends. Have a tissue handy for when your eyes start bleeding.

What's more, these methods didn't even exist until Java 1.5, so what goes for a valid identifier in Java has in fact
changed over the years. That's as much the Unicode Consortium's fault as anyone's, but still. Joy.

Speaking non-asymptotically, the index is neither as small nor as fast as I would like.
Avenues for future exploration would include:
  - Making large parts of the tree implicit, essentially reducing each node to a
    single 64-bit integer and the index to array plus a couple of side lookup tables
  - Using finite-state transduction rather than a trie
