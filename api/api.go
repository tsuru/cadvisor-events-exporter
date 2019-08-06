package api

import (
	"log"
	"net/http"

	info "github.com/google/cadvisor/info/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type EventLister interface {
	ListEvents(r *http.Request) ([]info.Event, error)
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
)

type server struct {
	lister EventLister
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	err := s.processEvents(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) processEvents(w http.ResponseWriter, r *http.Request) error {
	events, err := s.lister.ListEvents(r)
	if err != nil {
		return err
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

func eventsToMetrics(events []info.Event) ([]prometheus.Metric, error) {
	type metric struct {
		containerName string
		eventType     string
		oomProcess    string
	}
	counters := make(map[metric]float64)
	for _, evt := range events {
		key := metric{
			containerName: evt.ContainerName,
			eventType:     string(evt.EventType),
		}
		if evt.EventData.OomKill != nil {
			key.oomProcess = evt.EventData.OomKill.ProcessName
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

func RunServer(lister EventLister) {
	addr := "0.0.0.0:8888"
	log.Printf("Listening on %q\n", addr)
	http.ListenAndServe(addr, &server{
		lister: lister,
	})
}
