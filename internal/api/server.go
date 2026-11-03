package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"external-metrics-gateway/internal/provider"
)

type Server struct {
	router *provider.Router
}

func NewServer(router *provider.Router) *Server {
	return &Server{router: router}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/apis/external.metrics.k8s.io/v1beta1", s.discovery)
	mux.HandleFunc("/apis/external.metrics.k8s.io/v1beta1/", s.getMetric)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":         "APIResourceList",
		"apiVersion":   "v1",
		"groupVersion": "external.metrics.k8s.io/v1beta1",
		"resources": []map[string]any{
			{
				"name":         "*",
				"singularName": "",
				"namespaced":   true,
				"kind":         "ExternalMetricValueList",
				"verbs":        []string{"get"},
			},
		},
	})
}

func (s *Server) getMetric(w http.ResponseWriter, r *http.Request) {
	ns, metric, err := parseMetricPath(r.URL.Path)
	if err != nil {
		log.Printf("parse metric path %s failed: %v", r.URL.Path, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	list, err := s.router.Fetch(r.Context(), ns, metric, r.URL.Path, r.URL.Query())
	if err != nil {
		log.Printf("fetch metric %s/%s failed: %v", ns, metric, err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func parseMetricPath(path string) (namespace string, metric string, err error) {
	base := "/apis/external.metrics.k8s.io/v1beta1/namespaces/"
	if !strings.HasPrefix(path, base) {
		log.Printf("parse metric path %s failed: %v", path, err)
		return "", "", errors.New("invalid path")
	}
	rest := strings.TrimPrefix(path, base)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		log.Printf("parse metric path %s failed: %v", path, err)
		return "", "", errors.New("expected /namespaces/{namespace}/{metric}")
	}
	return parts[0], parts[1], nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"kind":       "Status",
		"apiVersion": "v1",
		"status":     "Failure",
		"reason":     http.StatusText(status),
		"message":    message,
		"code":       status,
	})
}
