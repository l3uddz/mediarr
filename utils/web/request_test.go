package web

import (
	"os"
	"testing"
)

/* Test Get Response Timeout */

func TestGetResponseTimeout(t *testing.T) {
	// send request
	resp, err := GetResponse(GET, "https://httpbin.davecheney.com/delay/5", 3)
	if err != nil && !os.IsTimeout(err) {
		t.Errorf("Expected timeout in 3 seconds but got error: %v", err)
		return
	} else if err == nil {
		defer DrainAndClose(resp.Response().Body)
		t.Errorf("Expected timeout in 3 seconds but got no error...")
	}
}
