package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TODO: rework ?
type Metrics struct {
	withdrawalsClaimed prometheus.Counter
	withdrawalsCounter uint64

	withdrawalAddresses prometheus.Gauge
	addressesCounter    uint64

	// Public RPC bad responses
	rpcErrors prometheus.Gauge

	registry *prometheus.Registry
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		withdrawalsClaimed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "autoclaimer",
			Name:      "withdrawals_claimed",
			Help:      "Number of withdrawals claimed on the last claim.",
		}),
		withdrawalAddresses: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "autoclaimer",
			Name:      "withdrawal_addresses",
			Help:      "Number of unique withdrawal addresess processed on the last claim.",
		}),
		rpcErrors: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "autoclaimer",
			Name:      "gnosis_rpc_errors",
			Help:      "Number of bad responses from public execution layer RPC.",
		}),
	}
	reg.MustRegister(m.withdrawalsClaimed, m.withdrawalAddresses, m.rpcErrors)
	m.registry = reg
	return m
}

// Commit called every time claimBatches executed correctly
// to update real proccessed values and not rely on intermediate state.
func (m *Metrics) Commit() {
	m.withdrawalsClaimed.Add(float64(m.withdrawalsCounter))
	m.withdrawalsCounter = 0

	m.withdrawalAddresses.Set(float64(m.addressesCounter))
	m.addressesCounter = 0
}

// wrapHandler needs to nullify rpcErrors metric after prometheus scrapes it.
// Its necessary to have updated rpcError value every time.
func (m *Metrics) wrapHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer m.rpcErrors.Set(0) // will be executed after promHandler handled the call
			h.ServeHTTP(w, r)        // call original prom handler
		},
	)
}

func (m *Metrics) serve(ctx context.Context) {
	promHandler := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	// TODO: make port configurable?
	server := &http.Server{Addr: ":8888", Handler: m.wrapHandler(promHandler)}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Error("metrics serve error: %s", err.Error())
		}
	}()

	// blocks untill parent context will be canceled
	<-ctx.Done()

	// gracefull shutdown
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(c); err != nil {
		log.Error("can't stop metrics server: %s", err.Error())
	}
}
