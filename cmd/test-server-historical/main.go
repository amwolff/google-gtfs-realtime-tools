package main

import (
	"log"
	"net/http"

	"github.com/amwolff/google-gtfs-realtime-tools/fetch"
	"github.com/amwolff/google-gtfs-realtime-tools/provider/historical"
)

func main() {
	p, err := historical.NewHistoricalProvider(1024, "./provider/historical/21.csv.gz")
	if err != nil {
		log.Fatalln(err)
	}

	h := fetch.NewWithCache(p)

	if err := http.ListenAndServe("localhost:8081", h); err != nil {
		log.Println(err)
	}
}
