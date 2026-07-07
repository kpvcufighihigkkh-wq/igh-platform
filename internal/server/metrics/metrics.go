package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "igh_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "igh_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}, []string{"method", "path"})

	EdgeEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "igh_edge_events_total",
		Help: "Total edge events received",
	}, []string{"edge_id", "event_type"})

	EdgeNodesOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "igh_edge_nodes_online",
		Help: "Number of online edge nodes",
	})

	ActiveAlarms = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "igh_active_alarms",
		Help: "Number of active alarms",
	})

	DoffingsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "igh_doffings_total",
		Help: "Total doffings processed",
	})

	BobbinsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "igh_bobbins_total",
		Help: "Total bobbins created",
	})

	PLCConnectionStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "igh_plc_connected",
		Help: "PLC connection status (1=connected, 0=disconnected)",
	}, []string{"edge_id", "device_code"})

	WarehouseOccupancy = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "igh_warehouse_occupancy",
		Help: "Warehouse slot occupancy",
	}, []string{"warehouse_code", "status"})
)
