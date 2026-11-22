package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"time"

	"e-bank/backend/internal/antifraud/models"
)

const (
	lookupTimeout    = 1500 * time.Millisecond
	requestTimeout   = 650 * time.Millisecond
	maxCacheEntries  = 4096
	cacheTTL         = time.Hour
	maxCacheAge      = 24 * time.Hour
	maxResponseBytes = 32 << 10
)

type Resolver struct {
	endpoint *url.URL
	client   *http.Client
	mu       sync.Mutex
	cache    map[netip.Addr]location
}

type location struct {
	latitude   float64
	longitude  float64
	resolvedAt time.Time
	country    string
	city       string
}

type providerResponse struct {
	Success   bool     `json:"success"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Country   string   `json:"country"`
	City      string   `json:"city"`
}

func New(endpoint string) (*Resolver, error) {
	if endpoint == "" {
		return &Resolver{}, nil
	}

	parsed, err := url.Parse(endpoint)
	if err != nil || parsed == nil || parsed.Hostname() == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("ANTIFRAUD_GEOIP_URL must be an absolute HTTP(S) provider URL")
	}

	return &Resolver{
		endpoint: parsed,
		client: &http.Client{
			Timeout: requestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cache: make(map[netip.Addr]location),
	}, nil
}

func (r *Resolver) Resolve(ctx context.Context, ip netip.Addr) models.Geo {
	ip = ip.Unmap()
	missing := models.Geo{Status: models.GeoMissing}
	if r.endpoint == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.Zone() != "" {
		return missing
	}
	if cached := r.cached(ip); time.Since(cached.resolvedAt) < cacheTTL {
		return cached.geo(models.GeoCached)
	}

	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	endpoint := r.endpoint.JoinPath(ip.String()).String()
	resolved, retry, ok := r.lookup(ctx, endpoint)
	if !ok && retry && ctx.Err() == nil {
		resolved, _, ok = r.lookup(ctx, endpoint)
	}
	if ok {
		r.store(ip, resolved)
		return resolved.geo(models.GeoFresh)
	}
	if cached := r.cached(ip); time.Since(cached.resolvedAt) <= maxCacheAge {
		return cached.geo(models.GeoStale)
	}

	return missing
}

func (r *Resolver) lookup(ctx context.Context, endpoint string) (location, bool, bool) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return location{}, false, false
	}
	response, err := r.client.Do(request)
	if err != nil {
		return location{}, true, false
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		retry := response.StatusCode == http.StatusRequestTimeout || response.StatusCode >= 500
		return location{}, retry, false
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return location{}, true, false
	}
	var data providerResponse
	if len(body) > maxResponseBytes || json.Unmarshal(body, &data) != nil ||
		!data.Success || data.Latitude == nil || data.Longitude == nil ||
		*data.Latitude < -90 || *data.Latitude > 90 ||
		*data.Longitude < -180 || *data.Longitude > 180 {
		return location{}, false, false
	}

	return location{
		latitude:   *data.Latitude,
		longitude:  *data.Longitude,
		resolvedAt: time.Now().UTC(),
		country:    data.Country,
		city:       data.City,
	}, false, true
}

func (r *Resolver) cached(ip netip.Addr) location {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cache[ip]
}

func (r *Resolver) store(ip netip.Addr, resolved location) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.cache[ip]; !exists && len(r.cache) == maxCacheEntries {
		for key := range r.cache {
			delete(r.cache, key)
			break
		}
	}
	r.cache[ip] = resolved
}

func (l location) geo(status string) models.Geo {
	return models.Geo{
		Status:     status,
		Latitude:   &l.latitude,
		Longitude:  &l.longitude,
		ResolvedAt: &l.resolvedAt,
		Country:    l.country,
		City:       l.city,
	}
}
