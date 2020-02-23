# Tools to interact with Google Transit APIs

## Overview

Live Transit Updates is a feature within Google Maps that provides users with realtime transit information.
GTFS Realtime is a data exchange format consumed by Google Transit APIs.
The code here aims to make it easier to push or make Google fetch such feed, enabling the feature.

## Getting started

The easiest way to start is to implement the [`FeedProvider`](https://github.com/amwolff/google-gtfs-realtime-tools/blob/master/provider/provider.go) interface.

### Push

```go
package main

import (
	"net/http"
	"os"

	"github.com/amwolff/google-gtfs-realtime-tools/oauth"
)

func main() {
	httpClient := &http.Client{}

	f, err := os.Open("a/path/to/client_secrets.JSON")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c, err := oauth.NewClient(
		httpClient,
		f,
		"a/path/to/where/cache/.tokens.JSON",
		oauth.DefaultTokenExchangeURL,
		"YOUR_AUTHORIZATION_CODE",
		oauth.DefaultFeedUploadURL)
	if err != nil {
		panic(err)
	}

	p := ADataSourceThatImplementsFeedProvider{...}

	if err := c.Run(p, "feed.pb", "YOUR_ALKALI_ACCOUNT_ID", "YOUR_REALTIME_FEED_ID"); err != nil {
		panic(err)
	}
}
```

You can also use `UploadFeedMessage` method to have more control over the process.

### Fetch

```go
package main

import (
	"log"
	"net/http"

	"github.com/amwolff/google-gtfs-realtime-tools/fetch"
)

func main() {
	p := ADataSourceThatImplementsFeedProvider{...}

	h := fetch.NewWithCache(p)

	if err := http.ListenAndServe(":http", h); err != nil {
		log.Println(err)
	}
	log.Println("Begin serving your feed on all interfaces at HTTP port")
}
```

Although *push* seems more modern and sophisticated I strongly encourage you to use the *fetch* model, especially if you are a public transportation agency.
This way not only Google can fetch realtime transit data but also people like me.
Opening your data creates opportunities to build better working cities and, of course, the world.

I hope the docs and examples speak for themselves but if anything seems unclear - feel free to contact me or open an issue here.
