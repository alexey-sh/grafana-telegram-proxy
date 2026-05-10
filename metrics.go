package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const Version = "dev"

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Inbound HTTP requests by handler, method, and response status.",
	}, []string{"handler", "method", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Inbound HTTP request latency in seconds, by handler.",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler"})

	telegramSends = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telegram_send_total",
		Help: "Outbound Telegram sendMessage calls by outcome.",
	}, []string{"outcome"})

	telegramSendDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "telegram_send_duration_seconds",
		Help:    "Latency of outbound Telegram sendMessage calls, by outcome.",
		Buckets: prometheus.DefBuckets,
	}, []string{"outcome"})

	serviceInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_info",
		Help: "Constant 1 with build/version labels.",
	}, []string{"version"})
)

func init() {
	serviceInfo.WithLabelValues(Version).Set(1)
	for _, outcome := range []string{"success", "transport_error", "api_error"} {
		telegramSends.WithLabelValues(outcome)
		telegramSendDuration.WithLabelValues(outcome)
	}
	for _, handler := range []string{"health", "webhook"} {
		httpRequestDuration.WithLabelValues(handler)
	}
	httpRequests.WithLabelValues("health", "GET", "200")
	httpRequests.WithLabelValues("webhook", "POST", "200")
}
