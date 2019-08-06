# cadvisor events exporter

[![Build Status](https://travis-ci.org/tsuru/cadvisor-events-exporter.svg?branch=master)](https://travis-ci.org/tsuru/cadvisor-events-exporter)

This is a prototype of a proxy server transforming cadvisor events into
prometheus counters. Tracking events as prometheus metrics can be useful for
exposing the number of oom's and oom kills in a cluster.

## Installing and running

### Proxy mode

```
$ go get github.com/tsuru/cadvisor-events-exporter/cmd/cadvisor-events-proxy

$ cadvisor-events-proxy

$ curl http://127.0.0.1:8888/proxy/10.134.13.19:9094
# HELP cadvisor_container_events_total cadvisor events
# TYPE cadvisor_container_events_total counter
cadvisor_container_events_total{event_type="containerCreation",id="/docker/7d236739bb2dc453fe36723627f9a67f94de890de588404119192636fcbf1034",oom_process=""} 1
cadvisor_container_events_total{event_type="containerDeletion",id="/docker/7d236739bb2dc453fe36723627f9a67f94de890de588404119192636fcbf1034",oom_process=""} 1
cadvisor_container_events_total{event_type="oom",id="/docker/4cab384d33850b04ab3cf489e0957219cdb52c5edb5b3c719fcde0505d5b7162",oom_process=""} 2
cadvisor_container_events_total{event_type="oomKill",id="/docker/4cab384d33850b04ab3cf489e0957219cdb52c5edb5b3c719fcde0505d5b7162",oom_process="node"} 2
```

### Local mode

```
$ go get github.com/tsuru/cadvisor-events-exporter/cmd/cadvisor-local-exporter

$ cadvisor-local-exporter

$ curl http://127.0.0.1:8888/metrics
# HELP cadvisor_container_events_total cadvisor events
# TYPE cadvisor_container_events_total counter
cadvisor_container_events_total{event_type="containerCreation",id="/docker/7d236739bb2dc453fe36723627f9a67f94de890de588404119192636fcbf1034",oom_process=""} 1
cadvisor_container_events_total{event_type="containerDeletion",id="/docker/7d236739bb2dc453fe36723627f9a67f94de890de588404119192636fcbf1034",oom_process=""} 1
cadvisor_container_events_total{event_type="oom",id="/docker/4cab384d33850b04ab3cf489e0957219cdb52c5edb5b3c719fcde0505d5b7162",oom_process=""} 2
cadvisor_container_events_total{event_type="oomKill",id="/docker/4cab384d33850b04ab3cf489e0957219cdb52c5edb5b3c719fcde0505d5b7162",oom_process="node"} 2
```

