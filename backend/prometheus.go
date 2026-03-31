package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CallProm() {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)
	http.HandleFunc("/ping", m.ping)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.ListenAndServe(":2122", nil)
}

type metrics struct {
	pingCounter prometheus.Counter
}

func (m *metrics) ping(w http.ResponseWriter, r *http.Request) {
	m.pingCounter.Inc()
	fmt.Fprintf(w, "pong")
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		pingCounter: promauto.With(reg).NewCounter(
			prometheus.CounterOpts{
				Name: "ping_request_count",
				Help: "I dont know whats happening",
			},
		),
	}
	return m
}
