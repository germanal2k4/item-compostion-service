package parser

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	validationTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "parser_validation_time_seconds",
			Help:    "Time taken for syntax validation",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"template_name"},
	)

	semanticCheckTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "parser_semantic_check_time_seconds",
			Help:    "Time taken for semantic validation",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"template_name"},
	)

	conversionTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "parser_conversion_time_seconds",
			Help:    "Time taken for converting to Go structures",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"template_name"},
	)

	syntaxErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parser_syntax_errors_total",
			Help: "Total number of syntax errors",
		},
		[]string{"template_name"},
	)

	semanticErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parser_semantic_errors_total",
			Help: "Total number of semantic errors",
		},
		[]string{"template_name"},
	)

	versionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parser_version_errors_total",
			Help: "Total number of version compatibility errors",
		},
		[]string{"template_name"},
	)

	conversionSpeed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "parser_conversion_speed_elements_per_second",
			Help: "Number of elements processed per second",
		},
		[]string{"template_name"},
	)

	memoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "parser_memory_usage_bytes",
			Help: "Memory usage of the parser",
		},
		[]string{"template_name"},
	)

	templateVersions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parser_template_versions_total",
			Help: "Distribution of template versions",
		},
		[]string{"template_name", "version"},
	)

	fieldUsageStats = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parser_field_usage_total",
			Help: "Frequency of field usage",
		},
		[]string{"template_name", "field_name"},
	)

	dependencyGraph = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "parser_dependency_relations_total",
			Help: "Relations between templates",
		},
		[]string{"template_name", "dependent_template"},
	)
)

var (
	metricsRegistered bool
	metricsMutex      sync.Mutex
)

func registerMetrics() {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	if metricsRegistered {
		return
	}

	prometheus.MustRegister(
		validationTime,
		semanticCheckTime,
		conversionTime,
		syntaxErrors,
		semanticErrors,
		versionErrors,
		conversionSpeed,
		memoryUsage,
		templateVersions,
		fieldUsageStats,
		dependencyGraph,
	)

	metricsRegistered = true
}

type MetricsCollector struct {
	templateName string
	startTime    time.Time
}

func NewMetricsCollector(templateName string) *MetricsCollector {
	registerMetrics()
	return &MetricsCollector{
		templateName: templateName,
		startTime:    time.Now(),
	}
}

func (m *MetricsCollector) RecordValidationTime() {
	validationTime.WithLabelValues(m.templateName).Observe(time.Since(m.startTime).Seconds())
}

func (m *MetricsCollector) RecordSemanticCheckTime() {
	semanticCheckTime.WithLabelValues(m.templateName).Observe(time.Since(m.startTime).Seconds())
}

func (m *MetricsCollector) RecordConversionTime() {
	conversionTime.WithLabelValues(m.templateName).Observe(time.Since(m.startTime).Seconds())
}

func (m *MetricsCollector) RecordSyntaxError() {
	syntaxErrors.WithLabelValues(m.templateName).Inc()
}

func (m *MetricsCollector) RecordSemanticError() {
	semanticErrors.WithLabelValues(m.templateName).Inc()
}

func (m *MetricsCollector) RecordVersionError() {
	versionErrors.WithLabelValues(m.templateName).Inc()
}

func (m *MetricsCollector) RecordConversionSpeed(elements int) {
	conversionSpeed.WithLabelValues(m.templateName).Set(float64(elements) / time.Since(m.startTime).Seconds())
}

func (m *MetricsCollector) RecordMemoryUsage(bytes int64) {
	memoryUsage.WithLabelValues(m.templateName).Set(float64(bytes))
}

func (m *MetricsCollector) RecordTemplateVersion(version string) {
	templateVersions.WithLabelValues(m.templateName, version).Inc()
}

func (m *MetricsCollector) RecordFieldUsage(fieldName string) {
	fieldUsageStats.WithLabelValues(m.templateName, fieldName).Inc()
}

func (m *MetricsCollector) RecordDependency(dependentTemplate string) {
	dependencyGraph.WithLabelValues(m.templateName, dependentTemplate).Inc()
}
