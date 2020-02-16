// Package dummy contains example implementation of the provider.FeedProvider.
package dummy

import (
	"time"

	transitrealtime "github.com/amwolff/google-gtfs-realtime-tools/gen/go"
	"github.com/golang/protobuf/proto"
)

// DummyProvider is an example implementation of the provider.FeedProvider that
// streams example data. It does not close the underlying channel.
type DummyProvider struct {
	c     chan<- *transitrealtime.FeedMessage
	Delay time.Duration
}

func (d *DummyProvider) GetFeedChannel() chan<- *transitrealtime.FeedMessage {
	return d.c
}

func (d *DummyProvider) Stream(feed chan<- *transitrealtime.FeedMessage) {
	d.c = feed

	fullDataset := transitrealtime.FeedHeader_FULL_DATASET
	scheduled := transitrealtime.TripDescriptor_SCHEDULED
	inTransitTo := transitrealtime.VehiclePosition_IN_TRANSIT_TO
	unknownCongestionLevel := transitrealtime.VehiclePosition_UNKNOWN_CONGESTION_LEVEL
	empty := transitrealtime.VehiclePosition_EMPTY
	for {
		feed <- &transitrealtime.FeedMessage{
			Header: &transitrealtime.FeedHeader{
				GtfsRealtimeVersion: proto.String("2.0"),
				Incrementality:      &fullDataset,
				Timestamp:           proto.Uint64(uint64(time.Now().Unix())),
			},
			Entity: []*transitrealtime.FeedEntity{
				{
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
						Timestamp:           proto.Uint64(uint64(time.Now().Unix())),
						CongestionLevel:     &unknownCongestionLevel,
						OccupancyStatus:     &empty,
					},
				},
			},
		}
		time.Sleep(d.Delay)
	}
}
