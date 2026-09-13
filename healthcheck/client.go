package healthcheck

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func checkServiceEndpoint(endpoint string) (bool, error) {
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	port := os.Getenv("GOCRAFT_HEALTHCHECK_PORT")
	if port == "" {
		port = "8080"
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%s%s", port, endpoint)
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, err
	}

	return true, nil
}

func CheckServiceEndpointAndExit(endpoint string) int {
	ok, err := checkServiceEndpoint(endpoint)
	if !ok || err != nil {
		return 1
	}

	return 0
}
