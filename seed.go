package main

import (
	"fmt"
	"net/http"
)

func CreateSeedMux(seeds []string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/seed", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Hit on seed from", req.RemoteAddr)
		for _, s := range seeds {
			rw.Write([]byte(fmt.Sprintln(s)))
		}
	})
	return mux
}

func StartSeedServer(addr string, mux *http.ServeMux) {
	http.ListenAndServe(fmt.Sprintf(addr), mux)
}
