package provider

import (
	transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"
)

type FeedMessageProvider interface {
	Provide() (feedMessage *transitrealtime.FeedMessage, err error)
}
