package provider

import transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"

// FeedProvider is the interface that wraps feed message streaming.
//
// Stream starts streaming GTFS-realtime dataset onto feed.
//
// It does not return errors nor errors should be included in the message.
// Implementations are encouraged to handle errors on their own. A standard
// scenario would be to not send any messages until problems (e.g. database
// connection problem) are resolved.
//
// Callers will typically concurrently call Stream only once.
//
// Implementations must close feed when they are done.
type FeedProvider interface {
	Stream(feed chan<- *transitrealtime.FeedMessage)
}
