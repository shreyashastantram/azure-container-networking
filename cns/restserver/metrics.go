package restserver

import (
	"net/http"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var httpRequestLatency = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_request_latency_seconds",
		Help: "Request latency in seconds by endpoint, verb, and response code.",
		//nolint:gomnd // default bucket consts
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1 ms to ~16 seconds
	},
	// TODO(rbtr):
	// there's no easy way to extract the HTTP response code from the response due to the
	// way the restserver is designed currently - but we should fix that and include "code" as
	// a label and value.
	[]string{"url", "verb"},
)

var ipAssignmentLatency = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name: "ip_assignment_latency_seconds",
		Help: "Pod IP assignment latency in seconds",
		//nolint:gomnd // default bucket consts
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1 ms to ~16 seconds
	},
)

var ipConfigStatusStateTransitionTime = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "ipconfigstatus_state_transition",
		Help: "Time spent by the IP Configuration Status in each state transition",
		//nolint:gomnd // default bucket consts
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1 ms to ~16 seconds
	},
	[]string{"previous_state", "next_state"},
)

func init() {
	metrics.Registry.MustRegister(
		httpRequestLatency,
		ipAssignmentLatency,
		ipConfigStatusStateTransitionTime,
	)
}

func newHandlerFuncWithHistogram(handler http.HandlerFunc, histogram *prometheus.HistogramVec) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		defer func() {
			histogram.WithLabelValues(req.URL.RequestURI(), req.Method).Observe(time.Since(start).Seconds())
		}()
		handler(w, req)
	}
}

func stateTransitionMiddleware(i *cns.IPConfigurationStatus, s types.IPState) {
	// if no state transition has been recorded yet, don't collect any metric
	if i.LastStateTransition.IsZero() {
		return
	}
	ipConfigStatusStateTransitionTime.WithLabelValues(string(i.GetState()), string(s)).Observe(time.Since(i.LastStateTransition).Seconds())
}
