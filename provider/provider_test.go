package provider

import (
	"log"
	"net/http"
	"time"

	"github.com/amwolff/google-gtfs-realtime-tools/fetch"
	"github.com/amwolff/google-gtfs-realtime-tools/provider/dummy"
)

func ExampleProvider() {
	p := dummy.NewDummyProvider(5 * time.Second)
	defer p.Close()

	h := fetch.NewWithCache(p)

	if err := http.ListenAndServe("localhost:http", h); err != nil {
		log.Println(err)
	}
}
