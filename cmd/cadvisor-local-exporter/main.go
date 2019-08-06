package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/events"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/tsuru/cadvisor-events-exporter/api"
	"k8s.io/klog"
)

func startManager() (manager.Manager, error) {
	klog.InitFlags(nil)
	flag.Parse()
	backendStorage, err := storage.New("")
	if err != nil {
		return nil, err
	}
	memoryStorage := memory.New(time.Minute, backendStorage)
	sysFs := sysfs.NewRealSysFs()
	containerManager, err := manager.New(
		memoryStorage,
		sysFs,
		60*time.Second,
		true,
		container.MetricSet{},
		http.DefaultClient,
		nil,
	)
	if err != nil {
		return nil, err
	}
	err = containerManager.Start()
	if err != nil {
		return nil, err
	}
	return containerManager, nil
}

type localEventsLister struct {
	manager manager.Manager
}

func (l *localEventsLister) ListEvents(r *http.Request) ([]info.Event, error) {
	req := events.NewRequest()
	req.IncludeSubcontainers = true
	req.EventType = map[info.EventType]bool{
		info.EventOom:               true,
		info.EventOomKill:           true,
		info.EventContainerCreation: true,
		info.EventContainerDeletion: true,
	}
	evts, err := l.manager.GetPastEvents(req)
	if err != nil {
		return nil, err
	}
	result := make([]info.Event, len(evts))
	for i, evt := range evts {
		result[i] = *evt
	}
	return result, nil
}

func run() error {
	m, err := startManager()
	if err != nil {
		return err
	}
	lister := &localEventsLister{
		manager: m,
	}
	api.RunServer(lister)
	return nil
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
