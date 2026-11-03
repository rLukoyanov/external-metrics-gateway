package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type KEDAProvider struct {
	baseURL string
	client  *http.Client
}

func NewKEDAProvider(baseURL string) *KEDAProvider {
	return &KEDAProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (k *KEDAProvider) Name() string { return "keda" }

func (k *KEDAProvider) Fetch(ctx context.Context, q Query) (ExternalMetricValueList, error) {
	list := ExternalMetricValueList{
		Kind:       "ExternalMetricValueList",
		APIVersion: "external.metrics.k8s.io/v1beta1",
		Items:      []ExternalMetricValue{},
	}
	if k.baseURL == "" {
		return list, fmt.Errorf("keda url is empty")
	}

	u, err := url.Parse(k.baseURL + q.OriginalPath)
	if err != nil {
		return list, fmt.Errorf("invalid keda url: %w", err)
	}
	u.RawQuery = q.QueryValues.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return list, fmt.Errorf("build request: %w", err)
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return list, fmt.Errorf("keda request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return list, fmt.Errorf("keda status: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return list, fmt.Errorf("decode keda response: %w", err)
	}
	if list.Kind == "" {
		list.Kind = "ExternalMetricValueList"
	}
	if list.APIVersion == "" {
		list.APIVersion = "external.metrics.k8s.io/v1beta1"
	}
	return list, nil
}
