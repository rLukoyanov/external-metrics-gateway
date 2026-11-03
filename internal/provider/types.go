package provider

import (
	"context"
	"net/url"
)

type Query struct {
	Namespace        string
	Metric           string
	Labels           map[string]string
	RawLabelSelector string
	OriginalPath     string
	QueryValues      url.Values
}

type ExternalMetricValue struct {
	MetricName   string            `json:"metricName"`
	MetricLabels map[string]string `json:"metricLabels,omitempty"`
	Timestamp    string            `json:"timestamp"`
	Value        string            `json:"value"`
}

type ExternalMetricValueList struct {
	Kind       string                `json:"kind"`
	APIVersion string                `json:"apiVersion"`
	Items      []ExternalMetricValue `json:"items"`
}

type Provider interface {
	Name() string
	Fetch(ctx context.Context, q Query) (ExternalMetricValueList, error)
}
