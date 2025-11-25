package metrics

import (
    "time"
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
    

    DBPoolTotalConns = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eratemanager_db_pool_total_conns",
            Help: "Total number of connections in the DB pool per driver",
        },
        []string{"driver"},
    )

    DBPoolIdleConns = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eratemanager_db_pool_idle_conns",
            Help: "Idle connections in the DB pool per driver",
        },
        []string{"driver"},
    )

    DBPoolAcquiredConns = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eratemanager_db_pool_acquired_conns",
            Help: "Currently acquired (in-use) connections per driver",
        },
        []string{"driver"},
    )

    DBPoolAcquiresTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "eratemanager_db_pool_acquires_total",
            Help: "Total number of connection acquires per driver",
        },
        []string{"driver"},
    )
)

func UpdateDBPoolMetrics(driver string, total, idle, acquired float64, acquires uint64) {
    DBPoolTotalConns.WithLabelValues(driver).Set(total)
    DBPoolIdleConns.WithLabelValues(driver).Set(idle)
    DBPoolAcquiredConns.WithLabelValues(driver).Set(acquired)
    DBPoolAcquiresTotal.WithLabelValues(driver).Add(float64(acquires))
}


var (
    ScheduledJobLastRun = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eratemanager_job_last_run_timestamp",
            Help: "Unix timestamp of the last completed run for a job",
        },
        []string{"job"},
    )

    ScheduledJobLastDurationSeconds = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "eratemanager_job_last_duration_seconds",
            Help: "Duration of the last completed run for a job",
        },
        []string{"job"},
    )

    ScheduledJobFailuresTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "eratemanager_job_failures_total",
            Help: "Total number of failed executions per job",
        },
        []string{"job"},
    )
)

func UpdateJobMetrics(job string, startedAt time.Time, err error) {
    dur := time.Since(startedAt).Seconds()
    ScheduledJobLastDurationSeconds.WithLabelValues(job).Set(dur)
    ScheduledJobLastRun.WithLabelValues(job).Set(float64(time.Now().Unix()))
    if err != nil {
        ScheduledJobFailuresTotal.WithLabelValues(job).Inc()
    }
}
