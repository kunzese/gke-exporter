package main

import (
	"log"
	"net/http"

	exporter "github.com/kunzese/gke-exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "0.0.1"

func main() {
	log.Printf("Starting gke-exporter (Version: %s)", version)

	gkeExporter := exporter.New()

	prometheus.MustRegister(gkeExporter)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Google Kubernetes Engine (GKE) Exporter</title></head>
             <body>
             <h1>Google Kubernetes Engine (GKE) Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
