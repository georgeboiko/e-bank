package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

func GetDatasource(service string) (string, error) {
	for _, suffix := range []string{"ADDR", "USER", "PASSWORD", "NAME"} {
		key := service + "_DB_" + suffix

		if os.Getenv(key) == "" {
			return "", fmt.Errorf("%s is required", key)
		}
	}

	dbURL := url.URL{
		Scheme: "postgres",
		Host:   os.Getenv(service + "_DB_ADDR"),
		User: url.UserPassword(
			os.Getenv(service+"_DB_USER"),
			os.Getenv(service+"_DB_PASSWORD"),
		),
		Path: "/" + os.Getenv(service+"_DB_NAME"),
	}

	return dbURL.String(), nil
}

func GetAddr(service string, port int) string {
	addr := os.Getenv(service + "_HTTP_ADDR")
	if addr == "" {
		addr = "localhost:" + strconv.Itoa(port)
	}
	return addr
}
