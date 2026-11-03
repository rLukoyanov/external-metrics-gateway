package main

import (
	"flag"
	"log"
	"net/http"

	"external-metrics-gateway/internal/api"
	"external-metrics-gateway/internal/config"
	"external-metrics-gateway/internal/provider"
)

func main() {
	cfgPath := flag.String("config", "./config.yaml", "path to config yaml")
	tlsCertFile := flag.String("tls-cert-file", "", "path to TLS certificate (enables HTTPS when set with key)")
	tlsKeyFile := flag.String("tls-key-file", "", "path to TLS private key (enables HTTPS when set with cert)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	providers := map[string]provider.Provider{}
	if cfg.Prometheus.URL != "" {
		providers["prometheus"] = provider.NewPrometheusProvider(cfg.Prometheus.URL)
	}
	if cfg.KEDA.URL != "" {
		providers["keda"] = provider.NewKEDAProvider(cfg.KEDA.URL)
	}

	router := provider.NewRouter(cfg.Routes, cfg.DefaultSource, providers)
	server := api.NewServer(router)

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: server.Handler(),
	}

	if *tlsCertFile != "" || *tlsKeyFile != "" {
		if *tlsCertFile == "" || *tlsKeyFile == "" {
			log.Fatal("both -tls-cert-file and -tls-key-file must be set")
		}
		log.Printf("starting external-metrics-gateway (HTTPS) on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServeTLS(*tlsCertFile, *tlsKeyFile); err != nil {
			log.Fatalf("server stopped: %v", err)
		}
		return
	}

	log.Printf("starting external-metrics-gateway (HTTP) on %s", cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
