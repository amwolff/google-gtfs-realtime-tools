// Package historical contains example implementation of the
// provider.FeedProvider.
package historical

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"
	"github.com/golang/protobuf/proto"
)

// HistoricalProvider is an example implementation of the provider.FeedProvider
// that streams historical data.
type HistoricalProvider struct {
	n    int
	data []*transitrealtime.FeedMessage
}

const (
	TripID              = 7
	NextTripID          = 19
	RouteID             = 4
	NextRouteID         = 21
	DirectionID         = 6
	NextDirectionID     = 23
	StartTime           = 18
	NextStartTime       = 20
	StartDate           = 1
	ID                  = 2
	Label               = 28
	NextLabel           = 29
	Latitude            = 12
	Longitude           = 11
	Bearing             = 30
	Odometer            = 10
	CurrentStopSequence = 8
	// CurrentStatus   = IN_TRANSIT_TO
	Timestamp = 1
	// CongestionLevel = VehiclePosition_UNKNOWN_CONGESTION_LEVEL
	// OccupancyStatus = ???
)

func getCurrentStopSequence(record []string) *uint32 {

}

func getOdometer(record []string) *float64 {

}

func getBearing(record []string) *float32 {

}

func getLongitude(record []string) *float32 {

}

func getLatitude(record []string) *float32 {

}

func getLabel(record []string) *string {

}

func getStartDate(record []string) *string {

}

func getStartTime(record []string) *string {

}

func getDirectionId(record []string) *uint32 {

}

func getRouteId(record []string) *string {

}

func getTripId(record []string) *string {

}

func getEntity(record []string) (*transitrealtime.FeedEntity, error) {
	s := transitrealtime.TripDescriptor_SCHEDULED
	c := transitrealtime.VehiclePosition_IN_TRANSIT_TO
	l := transitrealtime.VehiclePosition_UNKNOWN_CONGESTION_LEVEL

	t, err := strconv.ParseUint(record[Timestamp], 10, 64)
	if err != nil {
		return nil, err
	}

	return &transitrealtime.FeedEntity{
		Id: proto.String("vehicle-position-" + record[TripID]),
		Vehicle: &transitrealtime.VehiclePosition{
			Trip: &transitrealtime.TripDescriptor{
				TripId:               getTripId(record),
				RouteId:              getRouteId(record),
				DirectionId:          getDirectionId(record),
				StartTime:            getStartTime(record),
				StartDate:            getStartDate(record),
				ScheduleRelationship: &s,
			},
			Vehicle: &transitrealtime.VehicleDescriptor{
				Id:    proto.String(record[ID]),
				Label: getLabel(record),
			},
			Position: &transitrealtime.Position{
				Latitude:  getLatitude(record),
				Longitude: getLongitude(record),
				Bearing:   getBearing(record),
				Odometer:  getOdometer(record),
			},
			CurrentStopSequence: getCurrentStopSequence(record),
			CurrentStatus:       &c,
			Timestamp:           &t,
			CongestionLevel:     &l,
		},
	}, nil
}

// NewHistoricalProvider returns initialized instance of HistoricalProvider that
// pushes up to n messages and any error encountered. If n < 0 it will loop
// forever.
func NewHistoricalProvider(n int, pathToData string) (
	*HistoricalProvider,
	error) {

	f, err := os.Open(filepath.Clean(pathToData))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	csvReader := csv.NewReader(r)

	i := transitrealtime.FeedHeader_FULL_DATASET

	var data []*transitrealtime.FeedMessage
	for {
		m := &transitrealtime.FeedMessage{
			Header: &transitrealtime.FeedHeader{
				GtfsRealtimeVersion: proto.String("2.0"),
				Incrementality:      &i,
				Timestamp:           proto.Uint64(t),
			},
		}
		var e []*transitrealtime.FeedEntity
		for {
			record, err := csvReader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					goto ret
				}
				return nil, err
			}
		}
		data = append(data, m)
	}

ret:
	return &HistoricalProvider{
		n:    n,
		data: data,
	}, nil
}

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

func (h *HistoricalProvider) Stream(feed chan<- *transitrealtime.FeedMessage) {
	defer close(feed)
	for i := 0; h.n < 0 || h.n < i; i++ {
		var prev time.Time
		for _, m := range h.data {
			feed <- m
			curr := time.Unix(int64(m.GetEntity()[0].GetVehicle().GetTimestamp()), 0)
			time.Sleep(curr.Sub(prev))
			prev = curr
		}
	}
}
