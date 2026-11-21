package auth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type accessTokenClaims struct {
	Type        string `json:"typ"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func NewJWTMiddleware(issuer, audience string) (func(http.Handler) http.Handler, error) {
	issuer = strings.TrimSpace(issuer)
	issuerURL, err := url.Parse(issuer)
	if err != nil || issuerURL.Hostname() == "" ||
		(issuerURL.Scheme != "https" && issuerURL.Scheme != "http") ||
		issuerURL.User != nil || strings.ContainsAny(issuer, "?#") {
		return nil, errors.New("KEYCLOAK_ISSUER must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	host := issuerURL.Hostname()
	if issuerURL.Scheme == "http" && !strings.EqualFold(host, "localhost") && !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("KEYCLOAK_ISSUER must use HTTPS outside localhost")
	}

	audience = strings.TrimSpace(audience)
	if audience == "" {
		return nil, errors.New("KEYCLOAK_AUDIENCE is required")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	keyContext := oidc.ClientContext(context.Background(), client)
	jwksURL := strings.TrimRight(issuer, "/") + "/protocol/openid-connect/certs"
	keys := oidc.NewRemoteKeySet(keyContext, jwksURL)
	verifier := oidc.NewVerifier(issuer, keys, &oidc.Config{
		ClientID:             audience,
		SupportedSigningAlgs: []string{oidc.RS256},
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := r.Header.Values("Authorization")
			if len(headers) != 1 {
				writeUnauthorized(w)
				return
			}
			parts := strings.Fields(headers[0])
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w)
				return
			}
			token, err := verifier.Verify(r.Context(), parts[1])
			if err != nil || strings.TrimSpace(token.Subject) == "" {
				writeUnauthorized(w)
				return
			}
			var claims accessTokenClaims
			if err := token.Claims(&claims); err != nil || claims.Type != "Bearer" {
				writeUnauthorized(w)
				return
			}

			principal := Principal{Subject: token.Subject, Roles: claims.RealmAccess.Roles}
			next.ServeHTTP(w, r.WithContext(WithVerifiedPrincipal(r.Context(), principal)))
		})
	}, nil
}
