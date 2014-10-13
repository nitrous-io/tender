package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/nitrous-io/tender"
)

// current version of tender binary
var version string

var (
	databaseURL    string
	baseDomain     string
	tlsDomain      string
	freighterdPort int
	pubCACert      string
	pubCAKey       string
	privCACertPath string
)

// Configure require settings. Required env variables are:
// - ADDR: address that tender will listen to
// - DATABASE_URL: url of the postgresql main database
// - BASE_DOMAIN: base domain of the tender app i.e. .nitrousapp.com
// - FREIGHTERD_PORT: public freighted port. The freighterd server itself
// handles authentication so we dont need to handle it here
//
// TODO: eventually we may look into etcd ( and co ) so we dont have to touch
// the main database *because* latency
func main() {
	var (
		flVersion = flag.Bool("v", false, "Display the version and exit")
	)
	flag.Parse()

	if *flVersion {
		fmt.Printf("tender version: %s\n", version)
		os.Exit(0)
	}

	// setup env variables
	databaseURL = envStr("DATABASE_URL")
	freighterdPort = envInt("FREIGHTERD_PORT")
	baseDomain = envStr("BASE_DOMAIN")
	tlsDomain = envStr("TLS_DOMAIN")
	pubCACert = envStr("PUB_CRT")
	pubCAKey = envStr("PUB_KEY")
	privCACertPath = envStr("PRI_CRT")

	if baseDomain[0] != '.' {
		panic("BASE_DOMAIN should start with .")
	}

	crt, err := tls.LoadX509KeyPair(pubCACert, pubCAKey)
	if err != nil {
		log.Fatalf("tender: Failed to load SSL certificates: %q", err)
	}

	privCACert, err := ioutil.ReadFile(privCACertPath)
	if err != nil {
		log.Fatalf("tender: Failed to find private certificate: %s", privCACertPath)
	}

	httpProxy, err := tender.NewHTTPServer(baseDomain, databaseURL)
	if err != nil {
		log.Fatalf("tender: Failed to create http proxy %q", err)
	}

	httpsProxy, err := tender.NewServer(baseDomain, tlsDomain, databaseURL, freighterdPort, privCACert)
	if err != nil {
		log.Fatalf("tender: Failed to create https proxy, %q", err)
	}

	// create tls config
	tlsConfig := &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{crt},
	}

	// proxy port range for preview
	startPort, endPort := previewPorts()
	log.Printf("listen to port from %d to %d", startPort, endPort)
	for p := startPort; p <= endPort; p++ {
		go serveTLS(p, tlsConfig, httpProxy)
	}

	// proxy freighterd port
	log.Printf("listen to port %d", freighterdPort)
	serveTLS(freighterdPort, tlsConfig, httpsProxy)
}

func envStr(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("Missing env var " + name)
	}
	return v
}

func envInt(name string) int {
	v := os.Getenv(name)
	if v == "" {
		panic("Missing " + name)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("Invalid %s. Required integer, got %s", name, v))
	}
	return i
}

func previewPorts() (start, end int) {
	s := os.Getenv("PREVIEW_START_PORT")
	e := os.Getenv("PREVIEW_END_PORT")

	if os.Getenv("DEBUG") == "true" {
		start = 6666
		end = 6666
		return
	}

	if s == "" {
		start = 1024
	} else {
		sp, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("Invalid PREVIEW_START_PORT. Need integer, got %s", s))
		}
		start = sp
	}

	if e == "" {
		end = 9999
	} else {
		ep, err := strconv.Atoi(e)
		if err != nil {
			panic(fmt.Sprintf("Invalid PREVIEW_END_PORT. Need integer, got %s", e))
		}
		end = ep
	}

	return
}

func serveTLS(port int, tlsConfig *tls.Config, s http.Handler) {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:      addr,
		Handler:   s,
		TLSConfig: tlsConfig,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("tender: Failed to listen to %q, error: %q", addr, err)
	}

	ln = tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, tlsConfig)
	err = srv.Serve(ln)
	log.Fatalf("tender: Failed to serve %q, error: %q", addr, err)

}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
