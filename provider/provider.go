package provider

import (
	"fmt"
	"strings"
)

/* Public */

func Get(providerType string) (Interface, error) {
	switch strings.ToLower(providerType) {
	case "tvmaze":
		return NewTvMaze(), nil
	default:
		break
	}

	return nil, fmt.Errorf("unsupported media provider provided: %q", providerType)
}
