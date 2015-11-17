package drivers

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func enableDebugTransport() {
	http.DefaultTransport = &DebugTransport{
		Transport: http.DefaultTransport,
	}
}

type DebugTransport struct {
	Transport http.RoundTripper
}

func (t *DebugTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	log.Debugf("Send    ==>: %s %s", req.Method, req.URL)

	resp, err = t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	log.Debugf("Receive <==: %d %s (size=%d)", resp.StatusCode, resp.Request.URL, resp.ContentLength)

	return resp, err
}
