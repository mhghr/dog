package processor

import (
	"fmt"
	"strings"

	"monitoring-platform/packages/shared/domain"
)

// VMLabel is a single label in the VictoriaMetrics line protocol.
type VMLabel struct {
	Name  string
	Value string
}

// VMMetric is a single time series sample for VictoriaMetrics.
type VMMetric struct {
	Metric    string
	Labels    []VMLabel
	Value     float64
	Timestamp int64
}

// VMWriteRequest holds samples ready for the VictoriaMetrics import API.
type VMWriteRequest struct {
	Samples []VMMetric
}

// escapeLabelValue escapes a label value for Prometheus text exposition format.
// Backslash, double-quote and newline are escaped with a backslash prefix.
var escapeLabelValue = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	"\n", `\n`,
)

// ConvertToVM converts a domain metric batch into a VictoriaMetrics write request.
func ConvertToVM(batch domain.MetricBatch) VMWriteRequest {
	var req VMWriteRequest
	req.Samples = make([]VMMetric, 0, len(batch.Samples))
	for _, s := range batch.Samples {
		vm := VMMetric{
			Metric:    s.Name,
			Labels:    make([]VMLabel, 0, len(s.Labels)),
			Value:     s.Value,
			Timestamp: s.Timestamp.UnixMilli(),
		}
		for k, v := range s.Labels {
			vm.Labels = append(vm.Labels, VMLabel{Name: k, Value: v})
		}
		req.Samples = append(req.Samples, vm)
	}
	return req
}

// ToPrometheusText renders the request as Prometheus text exposition format,
// which is what VictoriaMetrics' /api/v1/import/prometheus endpoint accepts.
func (r VMWriteRequest) ToPrometheusText() string {
	var sb strings.Builder
	for _, s := range r.Samples {
		sb.WriteString(s.Metric)
		if len(s.Labels) > 0 {
			sb.WriteByte('{')
			for i, l := range s.Labels {
				if i > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString(fmt.Sprintf(`%s="%s"`, l.Name, escapeLabelValue.Replace(l.Value)))
			}
			sb.WriteByte('}')
		}
		sb.WriteString(fmt.Sprintf(" %g %d\n", s.Value, s.Timestamp))
	}
	return sb.String()
}
