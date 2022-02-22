package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	METRICS_PROVIDER_LABEL     = "provider"
	METRICS_CR_NAMESPACE_LABEL = "namespace"
	METRICS_CR_NAME_LABEL      = "name"
)

var (
	successfulGroupSyncs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "group_sync_successful_syncs_count",
			Help: "Number of Successful Synchronizations",
		},
		[]string{METRICS_PROVIDER_LABEL, METRICS_CR_NAMESPACE_LABEL, METRICS_CR_NAME_LABEL})

	unsuccessfulGroupSyncs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "group_sync_unsuccessful_syncs_count",
			Help: "Number of Unsuccessful Synchronizations",
		},
		[]string{METRICS_PROVIDER_LABEL, METRICS_CR_NAMESPACE_LABEL, METRICS_CR_NAME_LABEL})

	groupsSynchronized = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "group_sync_number_groups",
			Help: "Number of Groups Synchronized",
		},
		[]string{METRICS_PROVIDER_LABEL, METRICS_CR_NAMESPACE_LABEL, METRICS_CR_NAME_LABEL})

	groupsPruned = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "group_pruned_number_groups",
			Help: "Number of Groups Pruned",
		},
		[]string{METRICS_PROVIDER_LABEL, METRICS_CR_NAMESPACE_LABEL, METRICS_CR_NAME_LABEL})

	nextScheduledSynchronization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "group_sync_next_scheduled_sync",
			Help: "Time of Next Scheduled Synchronization",
		},
		[]string{METRICS_CR_NAMESPACE_LABEL, METRICS_CR_NAME_LABEL})

	groupSyncError = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "group_sync_error",
			Help: "Error Occurred During Group Synchronization",
		},
		[]string{METRICS_PROVIDER_LABEL, METRICS_CR_NAMESPACE_LABEL, METRICS_CR_NAME_LABEL})
)

func init() {
	metrics.Registry.MustRegister(successfulGroupSyncs, unsuccessfulGroupSyncs, groupsSynchronized, nextScheduledSynchronization, groupSyncError)
}
