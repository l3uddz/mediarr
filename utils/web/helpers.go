package web

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

/* Public */

func JoinURL(base string, paths ...string) string {
	// credits: https://stackoverflow.com/a/57220413
	p := path.Join(paths...)
	return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), strings.TrimLeft(p, "/"))
}

func DrainAndClose(rc io.ReadCloser) {
	_, _ = io.Copy(ioutil.Discard, rc)
	_ = rc.Close()
}
