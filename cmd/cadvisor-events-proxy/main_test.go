package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ListEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("all_events"))
		assert.Equal(t, "true", r.URL.Query().Get("subcontainers"))
		w.Write([]byte(`[{"container_name": "c1", "event_type": "containerCreate"}]`))
	}))
	defer srv.Close()
	commonExpected := []info.Event{
		{ContainerName: "c1", EventType: "containerCreate"},
	}
	srvURL, _ := url.Parse(srv.URL)
	tests := []struct {
		url         string
		expected    []info.Event
		expectedErr string
	}{
		{url: "/proxy/" + srv.URL, expected: commonExpected},
		{url: "/proxy/" + srvURL.Host, expected: commonExpected},
		{url: "/proxy/?url=" + srv.URL, expected: commonExpected},
		{url: "/proxy?url=" + srv.URL, expected: commonExpected},
		{url: "/proxy?url=" + srv.URL + "/", expected: commonExpected},
		{url: "/proxy?url=" + srvURL.Host, expected: commonExpected},
		{url: "/proxy?url=" + srvURL.Host + "/", expected: commonExpected},
		{url: "/proxy", expectedErr: `127.0.0.1:9094.*refused`},
		{url: "/proxy/", expectedErr: `127.0.0.1:9094.*refused`},
		{url: "/invalid/" + srv.URL, expectedErr: `not found`},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			lister := &proxyEventsLister{}
			request, err := http.NewRequest("GET", tt.url, nil)
			require.NoError(t, err)
			evts, err := lister.ListEvents(request)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Regexp(t, tt.expectedErr, err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, evts)
		})
	}
}
