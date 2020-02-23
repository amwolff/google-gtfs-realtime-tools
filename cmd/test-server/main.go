package main

import (
	"log"
	"net/http"
	"time"

	"github.com/amwolff/google-gtfs-realtime-tools/fetch"
	"github.com/amwolff/google-gtfs-realtime-tools/provider/dummy"
)

func main() { // go run -race main.go
	p := dummy.NewDummyProvider(5 * time.Second)
	go func() {
		time.Sleep(10 * time.Minute)
		log.Println("Closing DP")
		p.Close()
	}()

	h := fetch.NewWithCache(p)

	if err := http.ListenAndServe("localhost:8080", h); err != nil {
		log.Println(err)
	}
}
