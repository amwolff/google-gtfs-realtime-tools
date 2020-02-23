// Package fetch implements fetch model of delivering GTFS-realtime feed.
package fetch

import (
	"net/http"
	"sync"

	transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"
	"github.com/amwolff/google-gtfs-realtime-tools/provider"
	"github.com/golang/protobuf/proto"
)

// TODO: docs?
// TODO: make debug mode optional (this is debug mode now)

type WithCache struct {
	closed bool
	recent *transitrealtime.FeedMessage
	mu     sync.RWMutex
}

func (w *WithCache) preload(provider provider.FeedProvider) {
	feed := make(chan *transitrealtime.FeedMessage)

	go provider.Stream(feed)

	for {
		m, ok := <-feed
		if !ok {
			w.mu.Lock()
			w.closed = true
			w.mu.Unlock()
			return
		}
		w.mu.Lock()
		w.recent = m
		w.mu.Unlock()
	}
}

func NewWithCache(provider provider.FeedProvider) *WithCache {
	ret := &WithCache{recent: &transitrealtime.FeedMessage{}}

	go ret.preload(provider)

	return ret
}

const ise = http.StatusInternalServerError

func (w *WithCache) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	w.mu.RLock()
	if w.closed {
		http.Error(rw, http.StatusText(http.StatusTeapot), http.StatusTeapot)
		w.mu.RUnlock()
		return
	}
	m := proto.Clone(w.recent)
	w.mu.RUnlock()

	if err := proto.MarshalText(rw, m); err != nil {
		http.Error(rw, http.StatusText(ise), ise)
	}
}
