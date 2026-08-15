package report

import (
	"io"

	internal "gridflex/internal/report"
	"gridflex/model"
)

type Format = internal.Format
type Metrics = internal.Metrics

const Text = internal.Text
const JSON = internal.JSON

func ParseFormat(value string) (Format, error) {
	return internal.ParseFormat(value)
}

func Document(plan model.Plan, settled model.Settlement, audit model.AuditSnapshot) (model.ReportDocument, error) {
	return internal.Document(plan, settled, audit)
}

func Write(writer io.Writer, format Format, document model.ReportDocument) error {
	return internal.Write(writer, format, document)
}

func CalculateMetrics(plan model.Plan) Metrics {
	return internal.CalculateMetrics(plan)
}

func IntervalCostSeries(settled model.Settlement) []int64 {
	return internal.IntervalCostSeries(settled)
}

func IntervalCarbonSeries(settled model.Settlement) []int64 {
	return internal.IntervalCarbonSeries(settled)
}

func TargetCompletionPPM(plan model.Plan) int64 {
	return internal.TargetCompletionPPM(plan)
}
