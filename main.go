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

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type cadvisorEvent struct {
	ContainerName string            `json:"container_name"`
	Timestamp     time.Time         `json:"timestamp"`
	EventType     string            `json:"event_type"`
	EventData     cadvisorEventData `json:"event_data"`
}

type cadvisorEventData struct {
	OOM oomEventData `json:"oom"`
}

type oomEventData struct {
	Pid         int    `json:"pid"`
	ProcessName string `json:"process_name"`
}

type metric struct {
	containerName string
	eventType     string
	oomProcess    string
}

type prometheusCollector struct {
	metrics []prometheus.Metric
}

func (c *prometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- eventsDesc
}

func (c *prometheusCollector) Collect(ch chan<- prometheus.Metric) {
	for i := range c.metrics {
		ch <- c.metrics[i]
	}
}

var (
	eventsDesc = prometheus.NewDesc(
		"cadvisor_container_events_total",
		"cadvisor events",
		[]string{"id", "event_type", "oom_process"},
		nil,
	)

	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	httpsRegexp = regexp.MustCompile(`^https?://.*`)
	urlRegexp   = regexp.MustCompile(`/.+?/(.+)`)
)

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/proxy") {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	grafanaURL := parseGrafanaURL(r)
	err := processEvents(w, r, grafanaURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func processEvents(w http.ResponseWriter, r *http.Request, grafanaURL string) error {
	if !httpsRegexp.MatchString(grafanaURL) {
		grafanaURL = fmt.Sprintf("http://%s", grafanaURL)
	}
	u, err := url.Parse(grafanaURL)
	if err != nil {
		return err
	}
	u.Path = "/api/v1.3/events"
	qs := u.Query()
	qs.Set("all_events", "true")
	qs.Set("subcontainers", "true")
	u.RawQuery = qs.Encode()
	rsp, err := client.Get(u.String())
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	dec := json.NewDecoder(rsp.Body)
	var events []cadvisorEvent
	err = dec.Decode(&events)
	if err != nil {
		data, _ := ioutil.ReadAll(dec.Buffered())
		return errors.Wrapf(err, "unable to parse: %v", string(data))
	}
	metrics, err := eventsToMetrics(events)
	if err != nil {
		return err
	}
	collector := &prometheusCollector{metrics: metrics}
	reg := prometheus.NewRegistry()
	reg.Register(collector)
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError})
	handler.ServeHTTP(w, r)
	return nil
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

func eventsToMetrics(events []cadvisorEvent) ([]prometheus.Metric, error) {
	counters := make(map[metric]float64)
	for _, evt := range events {
		key := metric{
			containerName: evt.ContainerName,
			eventType:     evt.EventType,
			oomProcess:    evt.EventData.OOM.ProcessName,
		}
		counters[key]++
	}
	var metrics []prometheus.Metric
	for m, counter := range counters {
		metric, err := prometheus.NewConstMetric(eventsDesc, prometheus.CounterValue, counter,
			m.containerName, m.eventType, m.oomProcess)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func main() {
	http.ListenAndServe("0.0.0.0:8888", &server{})
}
