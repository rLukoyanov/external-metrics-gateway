package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"external-metrics-gateway/internal/config"
)

type Router struct {
	routes        []config.Route
	providers     map[string]Provider
	defaultSource string
}

func NewRouter(routes []config.Route, defaultSource string, providers map[string]Provider) *Router {
	return &Router{
		routes:        routes,
		providers:     providers,
		defaultSource: defaultSource,
	}
}

func (r *Router) Fetch(ctx context.Context, namespace, metric, path string, values url.Values) (ExternalMetricValueList, error) {
	source, mappedMetric := r.resolve(metric)
	p, ok := r.providers[source]
	if !ok {
		return ExternalMetricValueList{}, fmt.Errorf("provider %q not configured", source)
	}

	labels := parseLabelSelector(values.Get("labelSelector"))
	q := Query{
		Namespace:        namespace,
		Metric:           mappedMetric,
		Labels:           labels,
		RawLabelSelector: values.Get("labelSelector"),
		OriginalPath:     path,
		QueryValues:      values,
	}
	return p.Fetch(ctx, q)
}

func (r *Router) resolve(metric string) (source string, mappedMetric string) {
	for _, rt := range r.routes {
		if strings.HasPrefix(metric, rt.Prefix) {
			mapped := metric
			if rt.StripPrefix {
				mapped = strings.TrimPrefix(metric, rt.Prefix)
			}
			return rt.Source, mapped
		}
	}
	return r.defaultSource, metric
}

func parseLabelSelector(selector string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(selector) == "" {
		return out
	}
	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		if k != "" {
			out[k] = v
		}
	}
	return out
}
