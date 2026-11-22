package services

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"e-bank/backend/internal/user/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	loginPollInterval = 5 * time.Second
	loginPollOverlap  = time.Minute
	loginPageSize     = 100
)

type KeycloakLoginSource struct {
	httpClient  *http.Client
	tokenURL    string
	eventsURL   string
	clientID    string
	secret      string
	loginClient string
}

type keycloakLogin struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	ClientID  string `json:"clientId"`
	UserID    string `json:"userId"`
	IPAddress string `json:"ipAddress"`
	Time      int64  `json:"time"`
	Error     string `json:"error"`
}

func NewKeycloakLoginSource(issuer, realmURL, clientID, secret, loginClient string) (*KeycloakLoginSource, error) {
	issuer = strings.TrimRight(issuer, "/")
	_, issuerRealm, _ := strings.Cut(issuer, "/realms/")
	if realmURL == "" {
		realmURL = issuer
	}
	base, err := url.Parse(strings.TrimRight(realmURL, "/"))
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("invalid Keycloak events realm URL")
	}
	if base.Scheme != "https" {
		ip, err := netip.ParseAddr(base.Hostname())
		if base.Scheme != "http" || (base.Hostname() != "localhost" && (err != nil || !ip.IsLoopback())) {
			return nil, errors.New("Keycloak events realm URL must use HTTPS outside localhost")
		}
	}

	prefix, realm, ok := strings.Cut(base.Path, "/realms/")
	if !ok || realm == "" || strings.Contains(realm, "/") || realm != issuerRealm {
		return nil, errors.New("Keycloak events URL must use the same realm as KEYCLOAK_ISSUER")
	}
	tokenURL := base.JoinPath("protocol/openid-connect/token").String()
	base.Path = prefix + "/admin/realms/" + realm + "/events"
	base.RawPath = ""
	return &KeycloakLoginSource{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		tokenURL:    tokenURL,
		eventsURL:   base.String(),
		clientID:    clientID,
		secret:      secret,
		loginClient: loginClient,
	}, nil
}

func (s *KeycloakLoginSource) accessToken(ctx context.Context) (string, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.secret)
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := s.readJSON(req, &token); err != nil {
		return "", fmt.Errorf("obtain Keycloak events access token: %w", err)
	}
	if token.AccessToken == "" {
		return "", errors.New("Keycloak returned an empty access token")
	}
	return token.AccessToken, nil
}

func (s *KeycloakLoginSource) loginPage(ctx context.Context, token string, since, until time.Time, offset int) ([]keycloakLogin, error) {
	query := url.Values{
		"type":      {"LOGIN"},
		"client":    {s.loginClient},
		"direction": {"asc"},
		"dateTo":    {strconv.FormatInt(until.UnixMilli(), 10)},
		"first":     {strconv.Itoa(offset)},
		"max":       {strconv.Itoa(loginPageSize)},
	}
	if !since.IsZero() {
		query.Set("dateFrom", strconv.FormatInt(since.UnixMilli(), 10))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.eventsURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var logins []keycloakLogin
	if err := s.readJSON(req, &logins); err != nil {
		return nil, fmt.Errorf("read Keycloak login events: %w", err)
	}
	return logins, nil
}

func (s *KeycloakLoginSource) readJSON(req *http.Request, dest any) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(dest)
}

func (s *KeycloakLoginSource) Run(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) {
	ticker := time.NewTicker(loginPollInterval)
	defer ticker.Stop()

	var latest time.Time
	var ready bool
	for ctx.Err() == nil {
		var err error
		if !ready {
			latest, err = repositories.LatestLoginAt(ctx, pool)
			ready = err == nil
		}
		if err == nil {
			until := time.Now()
			if latest.After(until) {
				latest = until
			}
			since := latest
			if !since.IsZero() {
				since = since.Add(-loginPollOverlap)
			}
			err = s.recordLogins(ctx, logger, pool, since, until)
			if err == nil {
				latest = until
			}
		}
		if err != nil && ctx.Err() == nil {
			logger.Warn("Keycloak login poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *KeycloakLoginSource) recordLogins(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, since, until time.Time) error {
	token, err := s.accessToken(ctx)
	if err != nil {
		return err
	}
	for offset := 0; ; offset += loginPageSize {
		logins, err := s.loginPage(ctx, token, since, until, offset)
		if err != nil {
			return err
		}
		for _, login := range logins {
			if login.Type != "LOGIN" || login.Error != "" || login.ClientID != s.loginClient {
				continue
			}
			ip, err := netip.ParseAddr(login.IPAddress)
			if login.ID == "" || login.Time <= 0 || strings.TrimSpace(login.UserID) == "" ||
				strings.ContainsRune(login.UserID, 0) || err != nil || ip.Zone() != "" {
				logger.Warn("Skipping malformed Keycloak login event", "event_id", login.ID)
				continue
			}
			eventID := loginEventID(login.ID, login.Time)
			if err := repositories.RecordLogin(ctx, pool, eventID, login.UserID, ip.Unmap(), time.UnixMilli(login.Time)); err != nil {
				return fmt.Errorf("record Keycloak login %s: %w", login.ID, err)
			}
		}
		if len(logins) < loginPageSize {
			return nil
		}
	}
}

func loginEventID(sourceID string, occurredAtMillis int64) [16]byte {
	// Use the Keycloak event's time and ID to reproduce the same UUIDv7 on each poll.
	hash := sha256.Sum256([]byte(sourceID))
	var id [16]byte
	binary.BigEndian.PutUint64(id[:8], uint64(occurredAtMillis)<<16)
	copy(id[6:], hash[:10])
	id[6] = id[6]&0x0f | 0x70
	id[8] = id[8]&0x3f | 0x80
	return id
}
