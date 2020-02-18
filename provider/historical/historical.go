// Package historical contains example implementation of the
// provider.FeedProvider.
package historical

import (
	"time"

	transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"
)

type vehicleRecord struct {
	TripID              string `csv:"id_kursu"`
	NextTripID          string `csv:"nast_id_kursu"`
	RouteID             string `csv:"numer_lini"`
	NextRouteID         string `csv:"nast_num_lini"`
	DirectionID         string `csv:"kierunek"`
	NextDirectionID     string `csv:"nast_kierunek"`
	StartTime           string `csv:"plan_godz_rozp"`
	NextStartTime       string `csv:"nast_plan_godz_rozp"`
	StartDate           time.Time
	ID                  string  `csv:"nr_radia"`
	Label               string  `csv:"opis_tabl"`
	NextLabel           string  `csv:"nast_opis_tabl"`
	Latitude            float32 `csv:"szerokosc"`
	Longitude           float32 `csv:"dlugosc"`
	Bearing             float32 `csv:"wektor"`
	Odometer            float64 `csv:"droga_wyko"`
	CurrentStopSequence uint32  `csv:"lp_przyst"`
	// CurrentStatus   = IN_TRANSIT_TO
	Timestamp time.Time `csv:"ts"`
	// CongestionLevel = VehiclePosition_UNKNOWN_CONGESTION_LEVEL
	// OccupancyStatus = ???
}

// HistoricalProvider is an example implementation of the provider.FeedProvider
// that streams historical data.
type HistoricalProvider struct {
	c    chan<- *transitrealtime.FeedMessage
	d    time.Duration
	n    int
	data []*transitrealtime.FeedMessage
}

// NewHistoricalProvider returns initialized instance of HistoricalProvider that
// pushes up to n messages every d and any error encountered. If n < 0 it will
// loop forever.
func NewHistoricalProvider(d time.Duration, n int, pathToData string) (
	*HistoricalProvider,
	error) {

	return &HistoricalProvider{}, nil
}

func (h *HistoricalProvider) Stream(feed chan<- *transitrealtime.FeedMessage) {
	defer close(feed)
	for i := 0; h.n < 0 || h.n < i; i++ {
		for _, m := range h.data {
			feed <- m
			time.Sleep(h.d)
		}
	}
}
