// Package dummy contains example implementation of the provider.FeedProvider.
package dummy

import (
	"log"
	"os"
	"time"

	transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"
	"github.com/golang/protobuf/proto"
)

// DummyProvider is an example implementation of the provider.FeedProvider that
// streams example data. It does not close the underlying channel on its own.
type DummyProvider struct {
	l *log.Logger
	s chan struct{}
	d time.Duration
}

// NewDummyProvider returns DummyProvider that sends a message every d.
func NewDummyProvider(d time.Duration) DummyProvider {
	return DummyProvider{
		l: log.New(os.Stdout, "DummyProvider", log.LstdFlags),
		s: make(chan struct{}),
		d: d,
	}
}

func (d DummyProvider) Close() {
	d.s <- struct{}{}
}

func (d DummyProvider) Stream(feed chan<- *transitrealtime.FeedMessage) {
	defer close(feed)

	fullDataset := transitrealtime.FeedHeader_FULL_DATASET
	scheduled := transitrealtime.TripDescriptor_SCHEDULED
	inTransitTo := transitrealtime.VehiclePosition_IN_TRANSIT_TO
	unknownCongestionLevel := transitrealtime.VehiclePosition_UNKNOWN_CONGESTION_LEVEL
	empty := transitrealtime.VehiclePosition_EMPTY
	for {
		select {
		case <-d.s:
			return
		default:
			d.l.Println("Streaming another dummy FeedMessage")
		}
		unix := proto.Uint64(uint64(time.Now().Unix()))
		feed <- &transitrealtime.FeedMessage{
			Header: &transitrealtime.FeedHeader{
				GtfsRealtimeVersion: proto.String("2.0"),
				Incrementality:      &fullDataset,
				Timestamp:           unix,
			},
			Entity: []*transitrealtime.FeedEntity{
				{
					Id: proto.String("example-vehicle-position"),
					Vehicle: &transitrealtime.VehiclePosition{
						Trip: &transitrealtime.TripDescriptor{
							TripId:               proto.String("zjd5xAvO"),
							RouteId:              proto.String("gB0NV05e"),
							DirectionId:          proto.Uint32(0),
							StartTime:            proto.String("11:15:35"),
							StartDate:            proto.String("200229"),
							ScheduleRelationship: &scheduled,
						},
						Vehicle: &transitrealtime.VehicleDescriptor{
							Id:           proto.String("yLA6m0oM"),
							Label:        proto.String("70"),
							LicensePlate: proto.String("ZH726273"),
						},
						Position: &transitrealtime.Position{
							Latitude:  proto.Float32(47.377778),
							Longitude: proto.Float32(8.540278),
							Bearing:   proto.Float32(1.23456),
							Odometer:  proto.Float64(123456),
							Speed:     proto.Float32(13.89),
						},
						CurrentStopSequence: proto.Uint32(1),
						StopId:              proto.String("5bAyD0LO"),
						CurrentStatus:       &inTransitTo,
						Timestamp:           unix,
						CongestionLevel:     &unknownCongestionLevel,
						OccupancyStatus:     &empty,
					},
				},
			},
		}
		time.Sleep(d.d)
	}
}
