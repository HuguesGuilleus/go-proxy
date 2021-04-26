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
	"strings"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Hijack support
		h, ok := w.(http.Hijacker)
		if !ok {
			log.Println("Error: No hijack")
			http.Error(w, "Need Hijacker connexion", http.StatusInternalServerError)
			return
		} else if r.Method != http.MethodConnect {
			http.Error(w, "Need the HTTP method CONNECT", http.StatusMethodNotAllowed)
			return
		}

		// Connexion to the target
		a := r.URL.Host
		if !strings.ContainsRune(a, ':') {
			switch r.URL.Scheme {
			case "http":
				a += ":80"
			case "https":
				a += ":443"
			default:
				log.Println("Error: wrong url sheme")
				http.Error(w,
					fmt.Sprintf("Unknown the scheme %q (use http, https or print the port)", r.URL.Scheme),
					http.StatusBadRequest)
			}
		}
		target, err := net.Dial("tcp", a)
		if err != nil {
			log.Println("Error:", err)
			http.Error(w, fmt.Sprintf("Dial to %q fail: %v\n", a, err), http.StatusInternalServerError)
			return
		}
		defer target.Close()
		log.Println("Connexion", a)

		// Take the connexion
		client, _, err := h.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer client.Close()
		client.Write([]byte("200 Connexion Establish\r\n"))

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
	})

	l := flag.String("l", ":5000", "Listen address")
	flag.Parse()

	log.Println("Listen", *l)
	log.Fatal(http.ListenAndServe(*l, nil))
}
