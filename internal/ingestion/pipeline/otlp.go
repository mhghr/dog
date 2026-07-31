package pipeline

import (
	"fmt"
	"time"

	collector "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"

	"monitoring-platform/internal/domain"
)

// otlpResourceLabels are the resource attribute keys we copy onto metric labels.
var otlpResourceLabels = []string{
	"host.name",
	"os.type",
	"service.name",
	"service.instance.id",
}

// ConvertOTLPMetrics converts an OTLP metrics request into domain metric samples.
func ConvertOTLPMetrics(req *collector.ExportMetricsServiceRequest, receivedAt time.Time) []domain.MetricSample {
	var samples []domain.MetricSample

	for _, rm := range req.GetResourceMetrics() {
		resourceLabels := resourceAttrs(rm.GetResource())

		for _, sm := range rm.GetScopeMetrics() {
			for _, m := range sm.GetMetrics() {
				switch data := m.GetData().(type) {
				case *metricspb.Metric_Gauge:
					for _, dp := range data.Gauge.GetDataPoints() {
						if v := doubleValue(dp); v != nil {
							samples = append(samples, domain.MetricSample{
								Name:      m.GetName(),
								Type:      domain.MetricTypeGauge,
								Value:     *v,
								Labels:    mergeLabels(resourceLabels, attrMap(dp.GetAttributes())),
								Timestamp: time.Unix(0, int64(dp.GetTimeUnixNano())),
							})
						}
					}
				case *metricspb.Metric_Sum:
					for _, dp := range data.Sum.GetDataPoints() {
						if v := doubleValue(dp); v != nil {
							samples = append(samples, domain.MetricSample{
								Name:      m.GetName(),
								Type:      domain.MetricTypeSum,
								Value:     *v,
								Labels:    mergeLabels(resourceLabels, attrMap(dp.GetAttributes())),
								Timestamp: time.Unix(0, int64(dp.GetTimeUnixNano())),
							})
						}
					}
				}
			}
		}
	}

	return samples
}

func resourceAttrs(r *resourcepb.Resource) map[string]string {
	if r == nil {
		return nil
	}
	out := make(map[string]string)
	for _, kv := range r.GetAttributes() {
		for _, key := range otlpResourceLabels {
			if kv.GetKey() == key {
				out[normalizeLabelKey(kv.GetKey())] = attrValue(kv.GetValue())
			}
		}
	}
	return out
}

func attrMap(kvs []*commonpb.KeyValue) map[string]string {
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		out[normalizeLabelKey(kv.GetKey())] = attrValue(kv.GetValue())
	}
	return out
}

func normalizeLabelKey(k string) string {
	// OTLP uses dots (host.name); Prometheus labels prefer underscores.
	return NormalizeMetricName(k)
}

func attrValue(v *commonpb.AnyValue) string {
	if v == nil {
		return ""
	}
	switch x := v.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return x.StringValue
	case *commonpb.AnyValue_IntValue:
		return fmt.Sprintf("%d", x.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%f", x.DoubleValue)
	case *commonpb.AnyValue_BoolValue:
		return fmt.Sprintf("%t", x.BoolValue)
	default:
		return ""
	}
}

func doubleValue(dp *metricspb.NumberDataPoint) *float64 {
	switch v := dp.GetValue().(type) {
	case *metricspb.NumberDataPoint_AsDouble:
		return &v.AsDouble
	case *metricspb.NumberDataPoint_AsInt:
		f := float64(v.AsInt)
		return &f
	}
	return nil
}

func mergeLabels(ms ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range ms {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
