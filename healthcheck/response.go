package healthcheck

import (
	"encoding/json"
	"net/http"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

func writeInternalServerError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(msg))
}

func writeResponse(w http.ResponseWriter, res HealthCheckResponse) {
	bytes, err := json.Marshal(res)
	if err != nil {
		writeInternalServerError(w, "failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func writeOk(w http.ResponseWriter) {
	writeResponse(w, HealthCheckResponse{
		Status: "ok",
	})
}
