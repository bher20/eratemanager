package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    RequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "eratemanager_requests_total",
            Help: "Total number of requests per provider",
        },
        []string{"provider"},
    )

    RequestDurationSeconds = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "eratemanager_request_duration_seconds",
            Help:    "Request duration in seconds per provider and path",
            Buckets: prometheus.DefBuckets,
        },
        []string{"provider", "path"},
    )

    RequestErrorsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "eratemanager_request_errors_total",
            Help: "Total number of error responses per provider and path",
        },
        []string{"provider", "path", "code"},
    )
)
