package main

import (
	"os"

	"e-bank/backend/internal/user/keycloak"
)

func configureKeycloakAdmin() (*keycloak.Client, error) {
	clientID := os.Getenv("KEYCLOAK_EVENTS_CLIENT_ID")
	secret := os.Getenv("KEYCLOAK_EVENTS_CLIENT_SECRET")
	if clientID == "" && secret == "" {
		return nil, nil
	}
	return keycloak.NewClient(os.Getenv("KEYCLOAK_ISSUER"), os.Getenv("KEYCLOAK_EVENTS_REALM_URL"), clientID, secret)
}
