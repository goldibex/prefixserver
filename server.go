package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	index "github.com/goldibex/prefixserver/index"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
	"strings"
)

type result struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type resultsBuffer struct {
	values  [][]byte
	scores  []int
	results []result
}

func newResultsBuffer() *resultsBuffer {
	return &resultsBuffer{
		values:  make([][]byte, 10),
		scores:  make([]int, 10),
		results: make([]result, 10),
	}
}

var logger *log.Logger
var in *index.Index
var pool chan *resultsBuffer

func main() {

	concurrency := flag.Int("concurrency", 64, "Maximum number of responses to handle concurrently")
	profile := flag.Bool("pprof", false, "Enable pprof for server profiling")
	profileAddr := flag.String("pprof-addr", "localhost:6060", "TCP address to listen on for pprof server")
	addr := flag.String("addr", ":8080", "TCP address to listen on for Web server")
	tlsCertFile := flag.String("tls-cert", "", "Path to TLS certificate for server SSL")
	tlsKeyFile := flag.String("tls-key", "", "Path to TLS key for server SSL")

	flag.Parse()

	if flag.Arg(0) == "" {
		fmt.Fprintf(os.Stderr, "usage: %s index_file\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(2)
	}

	logger = log.New(os.Stderr, "[prefixserver] ", log.Ldate|log.Ltime|log.Lshortfile|log.LUTC)
	pool = make(chan *resultsBuffer, *concurrency)

	for i := 0; i < *concurrency-1; i++ {
		pool <- newResultsBuffer()
	}

	file, err := os.Open(flag.Arg(0))
	logger.Printf("Loading index from %s", flag.Arg(0))

	if err != nil {
		logger.Panicf("Loading %s: %s", flag.Arg(0), err)
	}

	in = index.New()

	dec := gob.NewDecoder(file)
	err = dec.Decode(in)
	if err != nil {
		logger.Panicf("Decoding %s: %s", flag.Arg(0), err)
	}
	logger.Printf("Index loaded.")

	if *profile {
		profileMux := http.NewServeMux()
		profileMux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		profileMux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		profileMux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		profileMux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		profileMux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		profileServer := http.Server{
			Addr:     *profileAddr,
			Handler:  profileMux,
			ErrorLog: logger,
		}
		go func() {
			logger.Printf("pprof available at %s", *profileAddr)
			log.Fatal(profileServer.ListenAndServe())
		}()
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handleHTTP))

	srv := http.Server{
		Addr:     *addr,
		ErrorLog: logger,
		Handler:  mux,
	}

	if *tlsCertFile != "" && *tlsKeyFile != "" {
		logger.Printf("Server listening at https://%s", *addr)
		log.Fatal(srv.ListenAndServeTLS(*tlsCertFile, *tlsKeyFile))
	} else {
		logger.Printf("Server listening at http://%s", *addr)
		log.Fatal(srv.ListenAndServe())
	}

}

func handleHTTP(w http.ResponseWriter, r *http.Request) {

	logger.Printf("(%s) %s %s", r.RemoteAddr, r.Method, r.URL.Path)

	resultsBuffer := <-pool
	defer func() {
		pool <- resultsBuffer
	}()
	// make sure everything is copacetic with content type
	accepts := strings.Split(r.Header.Get("Accept"), ",")
	accepted := accepts == nil

	if !accepted {
		for i := range accepts {
			if strings.HasPrefix(accepts[i], "*/*") || accepts[i] == "application/json" {
				accepted = true
				break
			}
		}
	}

	if !accepted {
		w.Header().Set("Accept", "application/json")
		logger.Printf("(%s) %d (client Accept: %s)", r.RemoteAddr, http.StatusNotAcceptable, r.Header.Get("Accept"))
		w.WriteHeader(http.StatusNotAcceptable)
		return
	} else if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		logger.Printf("(%s) %d (client method: %s)", r.RemoteAddr, http.StatusMethodNotAllowed, r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	prefix := path.Base(r.URL.Path)

	count := in.Find([]byte(prefix), resultsBuffer.values, resultsBuffer.scores)
	logger.Printf("(%s) %s: %d", r.RemoteAddr, prefix, count)

	if count == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`[]`))
		return
	} else {

		for i := 0; i < count; i++ {
			resultsBuffer.results[i].Name = string(resultsBuffer.values[i])
			resultsBuffer.results[i].Score = resultsBuffer.scores[i]
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(resultsBuffer.results[0:count]); err != nil {
			logger.Printf("While sending results for query %s: %s", prefix, err)
		}

	}

}
