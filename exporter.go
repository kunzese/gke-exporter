package exporter

import (
	"context"
	"sync"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
)

func contains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

type Exporter struct {
	Metrics map[string]*prometheus.Desc
}

func New() *Exporter {
	return &Exporter{
		Metrics: map[string]*prometheus.Desc{
			"gkeMasterVersion": prometheus.NewDesc(
				prometheus.BuildFQName("gke", "master", "version_count"),
				"Number of GKE clusters, partitioned by the version of their master node.",
				[]string{"version"},
				nil,
			),
			"gkeUnsupportedMasterVersion": prometheus.NewDesc(
				prometheus.BuildFQName("gke", "master", "unsupported_versions_count"),
				"Number of GKE clusters with unsupported master versions, partitioned by the location, project and version of their master node.",
				[]string{"version", "project_id", "project_name", "name", "location"},
				nil,
			),
		},
	}
}

func (e Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.Metrics["gkeMasterVersion"]
	ch <- e.Metrics["gkeUnsupportedMasterVersion"]
}

// Collect implements the prometheus.Collector interface.
func (e Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	containerService, err := container.NewService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	projectsListResponse, err := cloudresourcemanagerService.Projects.List().Filter("lifecycleState:ACTIVE").Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Found %d projects", len(projectsListResponse.Projects))

	var mutex = &sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(len(projectsListResponse.Projects))

	validMasterVersions := map[string][]string{}
	masterVersionCount := map[string]float64{}

	for _, p := range projectsListResponse.Projects {
		go func(p *cloudresourcemanager.Project) {
			defer wg.Done()
			resp, err := containerService.Projects.Locations.Clusters.List("projects/" + p.ProjectId + "/locations/-").Context(ctx).Do()
			if err != nil {
				if ae, ok := err.(*googleapi.Error); ok && ae.Code == http.StatusForbidden {
					log.Warnf("Missing roles/container.clusterViewer on %s (%s)", p.Name, p.ProjectId)
					return
				} else if ae, ok := err.(*googleapi.Error); ok && ae.Code == http.StatusTooManyRequests {
					log.Warn("Quota exceeded")
					return
				} else {
					log.Fatal(err)
				}
			}

			for _, c := range resp.Clusters {
				mutex.Lock()
				if _, ok := validMasterVersions[c.Location]; !ok {
					log.Infof("Pulling server configs for location %s", c.Location)
					serverConfig, err := containerService.Projects.Locations.GetServerConfig("projects/" + p.ProjectId + "/locations/" + c.Location).Do()
					if err != nil {
						if ae, ok := err.(*googleapi.Error); ok && ae.Code == http.StatusTooManyRequests {
							log.Warn("Quota exceeded")
							return
						} else {
							log.Fatal(err)
						}
					}

					validMasterVersions[c.Location] = serverConfig.ValidMasterVersions
				}

				if _, ok := masterVersionCount[c.CurrentMasterVersion]; !ok {
					masterVersionCount[c.CurrentMasterVersion] = 1
				} else {
					masterVersionCount[c.CurrentMasterVersion]++
				}
				mutex.Unlock()

				if !contains(c.CurrentMasterVersion, validMasterVersions[c.Location]) {
					ch <- prometheus.MustNewConstMetric(
						e.Metrics["gkeUnsupportedMasterVersion"],
						prometheus.CounterValue,
						1,
						[]string{
							c.CurrentMasterVersion,
							p.ProjectId,
							p.Name,
							c.Name,
							c.Location,
						}...,
					)
				}
			}
		}(p)
	}

	wg.Wait()

	for version, cnt := range masterVersionCount {
		ch <- prometheus.MustNewConstMetric(
			e.Metrics["gkeMasterVersion"],
			prometheus.CounterValue,
			cnt,
			[]string{
				version,
			}...,
		)
	}

	log.Info("Done")
}
