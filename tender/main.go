package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/nitrous-io/tender"
)

// current version of tender binary
var version string

var (
	databaseURL    string
	baseDomain     string
	tlsDomain      string
	freighterdPort int
	tenderPort     int
	pubCACert      string
	pubCAKey       string
	privCACertPath string
)

// Configure require settings. Required env variables are:
// - ADDR: address that tender will listen to
// - DATABASE_URL: url of the postgresql main database
// - BASE_DOMAIN: base domain of the tender app i.e. .nitrousapp.com
// - TENDER_PORT: port tender listens to.
// - FREIGHTERD_PORT: public freighted port. The freighterd server itself
// handles authentication so we dont need to handle it here
// - PUB_CRT: public facing ssl cert that we bought
// - PUB_KEY: the public facing ssl cert key
// - PRI_CRT: self-signed private ssl for intra comms
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
	tenderPort = envInt("TENDER_PORT")
	freighterdPort = envInt("FREIGHTERD_PORT")
	baseDomain = envStr("BASE_DOMAIN")
	tlsDomain = envStr("TLS_DOMAIN")
	pubCACert = envStr("PUB_CRT")
	pubCAKey = envStr("PUB_KEY")
	privCACertPath = envStr("PRI_CRT")

	if baseDomain[0] != '.' {
		panic("BASE_DOMAIN should start with .")
	}

	_, err := tls.LoadX509KeyPair(pubCACert, pubCAKey)
	if err != nil {
		log.Fatalf("tender: Failed to load SSL certificates: %s", err)
	}
	privCACert, err := ioutil.ReadFile(privCACertPath)
	if err != nil {
		log.Fatalf("tender: Failed to find private certificate: %s", privCACertPath)
	}

	s, err := tender.NewServer(baseDomain, tlsDomain, databaseURL, freighterdPort, privCACert)
	if err != nil {
		log.Fatalf("tender: Failed to start tender server, %v", err)
	}

	// proxy freighterd ports
	log.Printf("tender started, listening on port %d", tenderPort)
	err = http.ListenAndServeTLS(fmt.Sprintf(":%d", tenderPort), pubCACert, pubCAKey, s)
	if err != nil {
		log.Fatalf("tender: exit with error %s", err)
	}
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
