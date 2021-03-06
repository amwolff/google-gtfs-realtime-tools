// Package historical contains example implementation of the
// provider.FeedProvider.
package historical

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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
	l    *log.Logger
	n    int
	data []*transitrealtime.FeedMessage
}

const (
	tripID              = 7
	nextTripID          = 19
	routeID             = 4
	nextRouteID         = 21
	directionID         = 6
	nextDirectionID     = 23
	startTime           = 18
	nextStartTime       = 20
	startDate           = 1
	id                  = 2
	label               = 28
	nextLabel           = 29
	latitude            = 12
	longitude           = 11
	bearing             = 30
	odometer            = 10
	currentStopSequence = 8
	// CurrentStatus   = IN_TRANSIT_TO
	timestamp = 1
	// CongestionLevel = VehiclePosition_UNKNOWN_CONGESTION_LEVEL
	// OccupancyStatus = ???
)

func getCurrentStopSequence(record []string) (*uint32, error) {
	u, err := strconv.ParseUint(record[currentStopSequence], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("ParseUint: %w", err)
	}
	return proto.Uint32(uint32(u)), nil
}

func getOdometer(record []string) (*float64, error) {
	f, err := strconv.ParseFloat(record[odometer], 64)
	if err != nil {
		return nil, fmt.Errorf("ParseFloat: %w", err)
	}
	return proto.Float64(f), nil
}

func getBearing(record []string) (*float32, error) {
	f, err := strconv.ParseFloat(record[bearing], 32)
	if err != nil {
		return nil, fmt.Errorf("ParseFloat: %w", err)
	}

	if f < 0 {
		f = math.NaN()
	}

	return proto.Float32(float32(f)), nil
}

func getLongitude(record []string) (*float32, error) {
	f, err := strconv.ParseFloat(record[longitude], 32)
	if err != nil {
		return nil, fmt.Errorf("ParseFloat: %w", err)
	}
	return proto.Float32(float32(f)), nil
}

func getLatitude(record []string) (*float32, error) {
	f, err := strconv.ParseFloat(record[latitude], 32)
	if err != nil {
		return nil, fmt.Errorf("ParseFloat: %w", err)
	}
	return proto.Float32(float32(f)), nil
}

func getLabel(record []string) *string {
	if len(record[label]) > 0 {
		return proto.String(record[label])
	}
	return proto.String(record[nextLabel])
}

func getStartTime(record []string) *string {
	if len(record[startTime]) > 0 {
		return proto.String(record[startTime])
	}
	return proto.String(record[nextStartTime])
}

func convertDirection(d string) uint32 {
	if d == "T" {
		return 1
	}
	return 0
}

func getDirectionId(record []string) *uint32 {
	if len(record[directionID]) > 0 {
		return proto.Uint32(convertDirection(record[directionID]))
	}
	return proto.Uint32(convertDirection(record[nextDirectionID]))
}

func getRouteId(record []string) *string {
	if len(record[routeID]) > 0 {
		return proto.String(record[routeID])
	}
	return proto.String(record[nextRouteID])
}

func getTripId(record []string) string {
	if record[tripID] != "0" {
		return record[tripID]
	}
	return record[nextTripID]
}

func getEntity(record []string) (*transitrealtime.FeedEntity, error) {
	t, err := time.Parse("2006-01-02 15:04:05.999999-07", record[timestamp])
	if err != nil {
		return nil, fmt.Errorf("Parse: %w", err)
	}

	ts := uint64(t.Unix())

	lat, err := getLatitude(record)
	if err != nil {
		return nil, fmt.Errorf("getLatitude: %w", err)
	}
	lon, err := getLongitude(record)
	if err != nil {
		return nil, fmt.Errorf("getLongitude: %w", err)
	}
	vec, err := getBearing(record)
	if err != nil {
		return nil, fmt.Errorf("getBearing: %w", err)
	}
	odo, err := getOdometer(record)
	if err != nil {
		return nil, fmt.Errorf("getOdometer: %w", err)
	}
	seq, err := getCurrentStopSequence(record)
	if err != nil {
		return nil, fmt.Errorf("getCurrentStopSequence: %w", err)
	}

	s := transitrealtime.TripDescriptor_SCHEDULED
	var c transitrealtime.VehiclePosition_VehicleStopStatus
	if *seq == 0 {
		c = transitrealtime.VehiclePosition_STOPPED_AT
	} else {
		c = transitrealtime.VehiclePosition_IN_TRANSIT_TO
	}
	l := transitrealtime.VehiclePosition_UNKNOWN_CONGESTION_LEVEL

	return &transitrealtime.FeedEntity{
		Id: proto.String("vehicle-position-" + getTripId(record)),
		Vehicle: &transitrealtime.VehiclePosition{
			Trip: &transitrealtime.TripDescriptor{
				TripId:               proto.String(getTripId(record)),
				RouteId:              getRouteId(record),
				DirectionId:          getDirectionId(record),
				StartTime:            getStartTime(record),
				StartDate:            proto.String(t.Format("20060102")),
				ScheduleRelationship: &s,
			},
			Vehicle: &transitrealtime.VehicleDescriptor{
				Id:    proto.String(record[id]),
				Label: getLabel(record),
			},
			Position: &transitrealtime.Position{
				Latitude:  lat,
				Longitude: lon,
				Bearing:   vec,
				Odometer:  odo,
			},
			CurrentStopSequence: seq,
			CurrentStatus:       &c,
			Timestamp:           &ts,
			CongestionLevel:     &l,
		},
	}, nil
}

func getMessage(entities []*transitrealtime.FeedEntity) *transitrealtime.FeedMessage {
	i := transitrealtime.FeedHeader_FULL_DATASET
	t := entities[0].GetVehicle().GetTimestamp()
	return &transitrealtime.FeedMessage{
		Header: &transitrealtime.FeedHeader{
			GtfsRealtimeVersion: proto.String("2.0"),
			Incrementality:      &i,
			Timestamp:           &t,
		},
		Entity: entities,
	}
}

// NewHistoricalProvider returns initialized HistoricalProvider that pushes up
// to n times historical feed and any error encountered. If n < 0 it will loop
// forever.
func NewHistoricalProvider(n int, pathToData string) (
	*HistoricalProvider,
	error) {

	l := log.New(os.Stdout, "HistoricalProvider", log.LstdFlags)

	f, err := os.Open(filepath.Clean(pathToData))
	if err != nil {
		return nil, fmt.Errorf("Open: %w", err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("NewReader: %w", err)
	}
	defer r.Close()

	csvReader := csv.NewReader(r)
	if _, err := csvReader.Read(); err != nil { // Discard columns' names.
		return nil, fmt.Errorf("Read: %w", err)
	}

	var (
		data []*transitrealtime.FeedMessage
		pre  uint64
		aux  *transitrealtime.FeedEntity
	)
	for {
		var entities []*transitrealtime.FeedEntity
		for {
			if aux != nil {
				entities = append(entities, aux)
			}

			record, err := csvReader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					data = append(data, getMessage(entities))
					goto ret
				}
				return nil, fmt.Errorf("Read: %w", err)
			}

			entity, err := getEntity(record)
			if err != nil {
				return nil, fmt.Errorf("getEntity: %w", err)
			}

			cur := entity.GetVehicle().GetTimestamp()
			if cur != pre && pre > 0 {
				pre = cur
				aux = entity
				break
			}

			entities = append(entities, entity)

			pre = cur
			aux = nil
		}
		data = append(data, getMessage(entities))
	}

ret:
	l.Printf("Loaded %d messages", len(data))
	return &HistoricalProvider{
		l:    l,
		n:    n,
		data: data,
	}, nil
}

func (h *HistoricalProvider) Stream(feed chan<- *transitrealtime.FeedMessage) {
	defer close(feed)
	for i := 0; i < h.n || h.n < 0; i++ {
		var prev time.Time
		for _, m := range h.data {
			curr := time.Unix(int64(m.GetHeader().GetTimestamp()), 0)

			h.l.Printf("Serving message with stamp = %v", curr.UTC())
			feed <- m

			waitTime := curr.Sub(prev)
			if waitTime > time.Minute { // Time travel.
				time.Sleep(time.Minute)
			} else {
				time.Sleep(waitTime)
			}

			prev = curr
		}
	}
}
