package models

import "time"

const (
	GeoFresh   = "FRESH"
	GeoCached  = "CACHED"
	GeoStale   = "STALE"
	GeoMissing = "MISSING"
)

type Geo struct {
	Status     string
	Latitude   *float64
	Longitude  *float64
	ResolvedAt *time.Time
	Country    string
	City       string
}
