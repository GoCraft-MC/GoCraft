package server

import (
	"fmt"

	"GoCraft/java/handler"
)

func admissionError(name, address string) error {
	if !handler.IsWhitelisted(name) {
		return fmt.Errorf("you are not whitelisted on this server")
	}
	if reason, banned := handler.BanReason(name, address); banned {
		return fmt.Errorf("you are banned from this server: %s", reason)
	}
	return nil
}
