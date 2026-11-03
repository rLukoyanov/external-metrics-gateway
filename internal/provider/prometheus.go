package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PrometheusProvider struct {
	baseURL string
	client  *http.Client
}

func NewPrometheusProvider(baseURL string) *PrometheusProvider {
	return &PrometheusProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (p *PrometheusProvider) Name() string { return "prometheus" }

func (p *PrometheusProvider) Fetch(ctx context.Context, q Query) (ExternalMetricValueList, error) {
	list := ExternalMetricValueList{
		Kind:       "ExternalMetricValueList",
		APIVersion: "external.metrics.k8s.io/v1beta1",
		Items:      []ExternalMetricValue{},
	}

	expr := buildPromQL(q)
	if expr == "" {
		return list, nil
	}

	u, err := url.Parse(p.baseURL + "/api/v1/query")
	if err != nil {
		return list, fmt.Errorf("invalid prometheus url: %w", err)
	}
	v := u.Query()
	v.Set("query", expr)
	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return list, fmt.Errorf("build request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return list, fmt.Errorf("prometheus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return list, fmt.Errorf("prometheus status: %s", resp.Status)
	}

	var r struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return list, fmt.Errorf("decode prometheus response: %w", err)
	}
	if r.Status != "success" {
		if r.Error == "" {
			r.Error = "unknown error"
		}
		return list, fmt.Errorf("prometheus query failed: %s", r.Error)
	}

	for _, item := range r.Data.Result {
		if len(item.Value) < 2 {
			continue
		}
		ts, ok := asFloat(item.Value[0])
		if !ok {
			continue
		}
		valueStr, ok := item.Value[1].(string)
		if !ok {
			continue
		}
		f, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		list.Items = append(list.Items, ExternalMetricValue{
			MetricName:   q.Metric,
			MetricLabels: item.Metric,
			Timestamp:    time.Unix(int64(ts), int64((ts-math.Floor(ts))*float64(time.Second))).UTC().Format(time.RFC3339),
			Value:        formatQuantity(f),
		})
	}

	return list, nil
}

func buildPromQL(q Query) string {
	metric := strings.TrimSpace(q.Metric)
	if metric == "" {
		return ""
	}
	labels := map[string]string{}
	for k, v := range q.Labels {
		labels[k] = v
	}
	if _, ok := labels["namespace"]; !ok {
		labels["namespace"] = q.Namespace
	}

	if len(labels) == 0 {
		return metric
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}
	return fmt.Sprintf("%s{%s}", metric, strings.Join(parts, ","))
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func formatQuantity(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if math.Abs(v-math.Round(v)) < 1e-9 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strconv.FormatFloat(v, 'f', 6, 64)
}
