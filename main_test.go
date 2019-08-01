package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Server_ServeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("all_events"))
		assert.Equal(t, "true", r.URL.Query().Get("subcontainers"))
		w.Write([]byte(`[{"container_name": "c1", "event_type": "containerCreate"}]`))
	}))
	defer srv.Close()
	srvURL, _ := url.Parse(srv.URL)
	tests := []struct {
		url          string
		expectedCode int
		expectedBody string
	}{
		{url: "/proxy/" + srv.URL, expectedCode: 200, expectedBody: "cadvisor_container_events_total"},
		{url: "/proxy/" + srvURL.Host, expectedCode: 200},
		{url: "/proxy/?url=" + srv.URL, expectedCode: 200},
		{url: "/proxy?url=" + srv.URL, expectedCode: 200},
		{url: "/proxy?url=" + srv.URL + "/", expectedCode: 200},
		{url: "/proxy?url=" + srvURL.Host, expectedCode: 200},
		{url: "/proxy?url=" + srvURL.Host + "/", expectedCode: 200},
		{url: "/proxy", expectedCode: 500, expectedBody: `127.0.0.1:9094.*refused`},
		{url: "/proxy/", expectedCode: 500, expectedBody: `127.0.0.1:9094.*refused`},
		{url: "/invalid/" + srv.URL, expectedCode: 404},
	}
	for _, tt := range tests {
		s := &server{}
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest("GET", tt.url, nil)
		require.NoError(t, err)
		s.ServeHTTP(recorder, request)
		assert.Equal(t, tt.expectedCode, recorder.Code, recorder.Body.String())
		assert.Regexp(t, tt.expectedBody, recorder.Body.String())
	}
}

func Test_EventsToMetrics(t *testing.T) {
	tests := []struct {
		events   []cadvisorEvent
		expected []string
	}{
		{},
		{events: []cadvisorEvent{}},
		{
			events: []cadvisorEvent{
				{ContainerName: "c1", EventType: "oom", EventData: cadvisorEventData{OOM: oomEventData{Pid: 1, ProcessName: "a"}}},
				{ContainerName: "c2", EventType: "oom", EventData: cadvisorEventData{OOM: oomEventData{Pid: 1, ProcessName: "a"}}},
				{ContainerName: "c1", EventType: "oom", EventData: cadvisorEventData{OOM: oomEventData{Pid: 3, ProcessName: "a"}}},
				{ContainerName: "c1", EventType: "oom", EventData: cadvisorEventData{OOM: oomEventData{Pid: 1, ProcessName: "b"}}},
				{ContainerName: "c1", EventType: "containerCreate"},
			},
			expected: []string{
				`label:<name:"event_type" value:"containerCreate" > label:<name:"id" value:"c1" > label:<name:"oom_process" value:"" > counter:<value:1 > `,
				`label:<name:"event_type" value:"oom" > label:<name:"id" value:"c1" > label:<name:"oom_process" value:"a" > counter:<value:2 > `,
				`label:<name:"event_type" value:"oom" > label:<name:"id" value:"c1" > label:<name:"oom_process" value:"b" > counter:<value:1 > `,
				`label:<name:"event_type" value:"oom" > label:<name:"id" value:"c2" > label:<name:"oom_process" value:"a" > counter:<value:1 > `,
			},
		},
	}
	for _, tt := range tests {
		metrics, err := eventsToMetrics(tt.events)
		require.NoError(t, err)
		var dtoMetrics []string
		for _, m := range metrics {
			dto := new(dto.Metric)
			err = m.Write(dto)
			require.NoError(t, err)
			dtoMetrics = append(dtoMetrics, dto.String())
		}
		sort.Strings(dtoMetrics)
		sort.Strings(tt.expected)
		require.Equal(t, tt.expected, dtoMetrics)
	}
}
