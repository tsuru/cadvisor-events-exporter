package api

import (
	"sort"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func Test_EventsToMetrics(t *testing.T) {
	tests := []struct {
		events   []info.Event
		expected []string
	}{
		{},
		{events: []info.Event{}},
		{
			events: []info.Event{
				{ContainerName: "c1", EventType: "oom", EventData: info.EventData{OomKill: &info.OomKillEventData{Pid: 1, ProcessName: "a"}}},
				{ContainerName: "c2", EventType: "oom", EventData: info.EventData{OomKill: &info.OomKillEventData{Pid: 1, ProcessName: "a"}}},
				{ContainerName: "c1", EventType: "oom", EventData: info.EventData{OomKill: &info.OomKillEventData{Pid: 3, ProcessName: "a"}}},
				{ContainerName: "c1", EventType: "oom", EventData: info.EventData{OomKill: &info.OomKillEventData{Pid: 1, ProcessName: "b"}}},
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
