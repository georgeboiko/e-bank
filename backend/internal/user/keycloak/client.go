package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	http     *http.Client
	adminURL *url.URL
	tokenURL string
	clientID string
	secret   string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("Keycloak Admin API: HTTP %d", e.StatusCode)
}

func NewClient(issuer, realmURL, clientID, clientSecret string) (*Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("Keycloak Admin API requires KEYCLOAK_EVENTS_CLIENT_ID and KEYCLOAK_EVENTS_CLIENT_SECRET")
	}
	if realmURL == "" {
		realmURL = issuer
	}
	base, err := url.Parse(strings.TrimRight(realmURL, "/"))
	if err != nil || base.Hostname() == "" || base.User != nil || strings.ContainsAny(realmURL, "?#") ||
		(base.Scheme != "http" && base.Scheme != "https") {
		return nil, errors.New("KEYCLOAK_EVENTS_REALM_URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	host := base.Hostname()
	if base.Scheme == "http" && !strings.EqualFold(host, "localhost") && !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("KEYCLOAK_EVENTS_REALM_URL must use HTTPS outside localhost")
	}

	issuerURL, err := url.Parse(strings.TrimRight(issuer, "/"))
	if err != nil {
		return nil, errors.New("invalid KEYCLOAK_ISSUER")
	}
	_, issuerRealm, _ := strings.Cut(issuerURL.Path, "/realms/")
	prefix, realm, _ := strings.Cut(base.Path, "/realms/")
	if realm == "" || realm == "." || realm == ".." || strings.Contains(realm, "/") || realm != issuerRealm {
		return nil, errors.New("KEYCLOAK_EVENTS_REALM_URL must use the same realm as KEYCLOAK_ISSUER")
	}

	tokenURL := base.JoinPath("protocol", "openid-connect", "token").String()
	base.Path = prefix + "/admin/realms/" + realm
	base.RawPath = ""
	return &Client{
		http: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		adminURL: base,
		tokenURL: tokenURL,
		clientID: clientID,
		secret:   clientSecret,
	}, nil
}

func (c *Client) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}

	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.clientID, c.secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	start := time.Now()
	if err := c.read(req, &result); err != nil {
		return "", fmt.Errorf("obtain Keycloak service token: %v", err)
	}
	if result.AccessToken == "" {
		return "", errors.New("Keycloak returned an empty service token")
	}
	c.accessToken = result.AccessToken
	c.expiresAt = start.Add(time.Duration(result.ExpiresIn-10) * time.Second)
	return c.accessToken, nil
}

func (c *Client) do(ctx context.Context, method string, path []string, query url.Values, body, result any) error {
	if c == nil {
		return errors.New("Keycloak Admin API is not configured")
	}
	segments := make([]string, len(path))
	for i, segment := range path {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("invalid Keycloak Admin API path")
		}
		segments[i] = url.PathEscape(segment)
	}
	endpoint := c.adminURL.JoinPath(segments...)
	endpoint.RawQuery = query.Encode()

	var content io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		content = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), content)
	if err != nil {
		return err
	}
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.read(req, result)
}

func (c *Client) read(req *http.Request, result any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &StatusError{StatusCode: res.StatusCode}
	}
	if result == nil {
		return nil
	}

	const maxResponseBytes = 2 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxResponseBytes {
		return errors.New("Keycloak Admin API response is too large")
	}
	return json.Unmarshal(body, result)
}
