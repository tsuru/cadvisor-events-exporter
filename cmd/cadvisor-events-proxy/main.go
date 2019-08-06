package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/pkg/errors"
	"github.com/tsuru/cadvisor-events-exporter/api"
)

var (
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	httpsRegexp = regexp.MustCompile(`^https?://.*`)
	urlRegexp   = regexp.MustCompile(`/.+?/(.+)`)
)

type proxyEventsLister struct{}

func (s *proxyEventsLister) ListEvents(r *http.Request) ([]info.Event, error) {
	if !strings.HasPrefix(r.URL.Path, "/proxy") {
		return nil, errors.New("not found")
	}
	grafanaURL := parseGrafanaURL(r)
	if !httpsRegexp.MatchString(grafanaURL) {
		grafanaURL = fmt.Sprintf("http://%s", grafanaURL)
	}
	u, err := url.Parse(grafanaURL)
	if err != nil {
		return nil, err
	}
	u.Path = "/api/v1.3/events"
	qs := u.Query()
	qs.Set("all_events", "true")
	qs.Set("subcontainers", "true")
	u.RawQuery = qs.Encode()
	rsp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	dec := json.NewDecoder(rsp.Body)
	var events []info.Event
	err = dec.Decode(&events)
	if err != nil {
		data, _ := ioutil.ReadAll(dec.Buffered())
		return nil, errors.Wrapf(err, "unable to parse: %v", string(data))
	}
	return events, nil
}

func parseGrafanaURL(r *http.Request) string {
	grafanaURL := r.FormValue("url")
	if grafanaURL != "" {
		return grafanaURL
	}
	subMatch := urlRegexp.FindStringSubmatch(r.URL.Path)
	if len(subMatch) == 2 {
		grafanaURL = subMatch[1]
	}
	if grafanaURL != "" {
		return grafanaURL
	}
	return "http://127.0.0.1:9094"
}

func main() {
	lister := &proxyEventsLister{}
	api.RunServer(lister)
}
