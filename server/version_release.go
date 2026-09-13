//go:build release

package server

import (
	"fmt"
)

func GetVersion() string {
	if Version[0] >= '0' && Version[0] <= '0' {
		return fmt.Sprintf("v%s", Version)
	}

	return fmt.Sprintf("%s", Version)
}
