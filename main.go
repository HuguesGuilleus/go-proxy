// go proxy
// BSD 3-Clause License
// Copyright (c) 2021, Hugues GUILLEUS All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		log.Println("Request", r.Method, r.URL)
		rep, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			log.Println("Error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		header := w.Header()
		for k, vv := range rep.Header {
			for _, v := range vv {
				header.Add(k, v)
			}
		}
		w.WriteHeader(rep.StatusCode)
		io.Copy(w, rep.Body)
	} else {
		h, ok := w.(http.Hijacker)
		if !ok {
			log.Println("Error: No hijack")
			http.Error(w, "Need Hijacker connexion", http.StatusInternalServerError)
			return
		}

		// Connexion to the target
		target, err := net.Dial("tcp", r.URL.Host)
		if err != nil {
			log.Println("Error:", err)
			http.Error(w, fmt.Sprintf("Dial to %q fail: %v\n", r.URL.Host, err), http.StatusInternalServerError)
			return
		}
		defer target.Close()
		log.Println("Connexion", r.URL.Host)

		// Take the connexion
		client, _, err := h.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer client.Close()
		client.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

		// Bidiretionnal copy
		done := make(chan struct{}, 1)
		go func() {
			io.Copy(client, target)
			done <- struct{}{}
		}()
		go func() {
			io.Copy(target, client)
			done <- struct{}{}
		}()
		<-done
	}
}

func main() {
	l := flag.String("l", ":5000", "Listen address")
	flag.Parse()

	log.Println("Listen", *l)
	log.Fatal(http.ListenAndServe(*l, handler))
}
