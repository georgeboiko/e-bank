package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func WriteProblem(w http.ResponseWriter, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{
		Type:   "urn:ebank:problem:" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		Title:  title,
		Status: status,
		Code:   code,
		Detail: detail,
	})
}
