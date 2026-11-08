package httpserver

import (
	"net/http"

	"github.com/go-chi/render"
)

func LiveHandler(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{
		"status": "UP",
	})
}
