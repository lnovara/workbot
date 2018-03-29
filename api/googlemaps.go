package api

import (
	"context"
	"time"

	"googlemaps.github.io/maps"
)

var (
	mapsClient *maps.Client
)

// NewMapsClient initialize a client for the Google Maps API
func NewMapsClient(mapsAPIKey string) error {
	var err error
	mapsClient, err = maps.NewClient(maps.WithAPIKey(mapsAPIKey))
	return err
}

func timezone(lat float64, lng float64) (string, error) {
	r, err := mapsClient.Timezone(context.TODO(), &maps.TimezoneRequest{
		Location: &maps.LatLng{
			Lat: lat,
			Lng: lng,
		},
		Timestamp: time.Now(),
		Language:  "it",
	})
	if err != nil {
		return "", err
	}
	return r.TimeZoneID, nil
}
