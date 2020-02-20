package main

import (
	"github.com/amwolff/google-gtfs-realtime-tools/provider/historical"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	h, err := historical.NewHistoricalProvider(-1, "./provider/historical/21.csv.gz")
	if err != nil {
		panic(err)
	}
	spew.Dump(h)
}
