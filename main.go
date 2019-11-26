package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// tunnel the connections
func handleTunneling(resp http.ResponseWriter, req *http.Request) {
	log.Print("handleTunneling")
	txConn, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusServiceUnavailable)
		return
	}
	resp.WriteHeader(http.StatusOK)
	hijacker, ok := resp.(http.Hijacker)
	if !ok {
		http.Error(resp, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	rxConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusServiceUnavailable)
	}

	// spin up goroutines to exchange data
	go transfer(txConn, rxConn)
	go transfer(rxConn, txConn)
}

// go routine for two-way data transfer
func transfer(tx io.WriteCloser, rx io.ReadCloser) {
	defer tx.Close()
	defer rx.Close()
	io.Copy(tx, rx)
}

func handleHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Print("handleHTTP")
	pxyresp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer pxyresp.Body.Close()
	copyHeader(resp.Header(), pxyresp.Header)
	resp.WriteHeader(pxyresp.StatusCode)
	io.Copy(resp, pxyresp.Body)
}

// header copy helper
func copyHeader(tx, rx http.Header) {
	for k, vv := range rx {
		for _, v := range vv {
			tx.Add(k, v)
		}
	}
}

func main() {
	var pemPath string
	flag.StringVar(&pemPath, "cert", "server.crt", "path to pem file")

	var keyPath string
	flag.StringVar(&keyPath, "key", "server.key", "path to key file")

	var proto string
	flag.StringVar(&proto, "proto", "https", "Proxy protocol (http or https)")

	flag.Parse()

	if proto != "http" && proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				handleTunneling(resp, req)
			} else {
				handleHTTP(resp, req)
			}
		}),
		// Disable HTTP/2
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if proto == "http" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(pemPath, keyPath))
	}
}
