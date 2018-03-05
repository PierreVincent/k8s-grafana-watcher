package main

import (
	"flag"
	"log"
	"os"
	"time"
	"gopkg.in/cheggaaa/mb.v1"

	kapi "k8s.io/kubernetes/pkg/api"
	kcache "k8s.io/kubernetes/pkg/client/cache"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kframework "k8s.io/kubernetes/pkg/controller/framework"
	kselector "k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/util/wait"
	"fmt"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configmapDashboardAnnotation  = flag.String("dashboardAnnotation", os.Getenv("CONFIG_MAP_DASHBOARD_ANNOTATION"), "Annotation that states that this configmap contains a grafana dashboard")
	configmapDatasourceAnnotation = flag.String("dataSourceAnnotation", os.Getenv("CONFIG_MAP_DATASOURCE_ANNOTATION"), "Annotation that states that this configmap contains a grafana datasource")
	grafanaUrl      = flag.String("grafanaUrl", os.Getenv("GRAFANA_URL"), "Grafana URL to push dashboard updates")
	grafanaUsername = flag.String("grafanaUser", os.Getenv("GRAFANA_USERNAME"), "Grafana username to push dashboard updates")
	grafanaPassword = flag.String("grafanaPassword", os.Getenv("GRAFANA_PASSWORD"), "Grafana password to push dashboard updates")
	batchTime = flag.Int("batchtime", 5, "Time window to batch updates (in seconds, default: 5)")
	helpFlag         = flag.Bool("help", false, "")

	grafana = NewGrafanaUpdater(*grafanaUrl, *grafanaUsername, *grafanaPassword)

	configMapUpdates = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grafana_watcher_configmap_updates",
			Help: "Total number of updates per configmap",
		},
		[]string{"name", "namespace", "status"},
	)
)

func main() {
	prometheus.MustRegister(configMapUpdates)
	flag.Parse()

	if *helpFlag ||
		*configmapDashboardAnnotation == "" ||
		*configmapDatasourceAnnotation == "" ||
		*grafanaUrl == "" ||
		*grafanaUsername == "" ||
		*grafanaPassword == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	log.Printf("Grafana Watcher loaded.\n")
	log.Printf("ConfigMap dashboard annotation: %s\n", *configmapDashboardAnnotation)
	log.Printf("ConfigMap datasource annotation: %s\n", *configmapDatasourceAnnotation)

	// create client
	var kubeClient *kclient.Client
	kubeClient, err := kclient.NewInCluster()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	updateQ := mb.New(50)
	defer updateQ.Close()

	go updateWorker(updateQ, kubeClient)

	_ = watchForConfigmaps(kubeClient, func(interface{}) {
		updateQ.Add(".")
	})

	defer func() {
		log.Printf("Cleaning up.")
	}()

	// Start the Prometheus listener
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)
}

func waitForGrafanaUp() {
	grafanaHealthUrl := fmt.Sprintf("%s/api/health", *grafanaUrl)
	for {
		resp, err := http.Get(grafanaHealthUrl)
		grafanaUp := false
		if err != nil {
			log.Printf("Failed to request Grafana Health: %s", err)
		} else if resp.StatusCode != 200 {
			log.Printf("Grafana Health returned with %d", resp.StatusCode)
		} else {
			grafanaUp = true
		}

		if grafanaUp {
			break
		} else {
			log.Println("Trying Grafana Health again in 60s")
			time.Sleep(60 * time.Second)
		}
	}
}

func updateWorker(q *mb.MB, kubeClient *kclient.Client) {
	waitForGrafanaUp()

	dashboardsLookup := NewConfigMapLookup(*configmapDashboardAnnotation)
	datasourceLookup := NewConfigMapLookup(*configmapDatasourceAnnotation)
	log.Printf("Worker started")
	for {
		time.Sleep(time.Second * time.Duration(*batchTime))
		msgs := q.Wait()
		if len(msgs) == 0 {
			break
		}

		log.Printf("Worker processing %d updates", len(msgs))
		updateDatasources(datasourceLookup, kubeClient)
		updateDashboards(dashboardsLookup, kubeClient)
	}
	log.Printf("Worker closed")
}

func updateDatasources(datasourcesLookup *ConfigMapLookup, kubeClient *kclient.Client) {
	log.Println("Looking for datasources...")

	datasources := datasourcesLookup.FindNewEntries(kubeClient)
	log.Printf("Found %d datasources updates", len(datasources))

	for _, datasource := range datasources {
		err := refreshDatasource(&datasource)
		if err != nil {
			log.Printf("%s", err)
		}
	}
}

func refreshDatasource(datasource *ConfigMapEntry) error {
	log.Printf("Refreshing datasource: %s", datasource.Key)
	err := grafana.PushDatasource(datasource.Value)
	if err != nil {
		configMapUpdates.WithLabelValues(datasource.Name, datasource.Namespace, "failed").Inc()
		return err
	}
	configMapUpdates.WithLabelValues(datasource.Name, datasource.Namespace, "success").Inc()
	return nil
}

func updateDashboards(dashboardsLookup *ConfigMapLookup, kubeClient *kclient.Client) {
	log.Println("Looking for dashboards...")

	dashboards := dashboardsLookup.FindNewEntries(kubeClient)
	log.Printf("Found %d dashboard updates", len(dashboards))

	for _, dashboard := range dashboards {
		err := refreshDashboard(&dashboard)
		if err != nil {
			log.Printf("%s", err)
		}
	}
}

func refreshDashboard(dashboard *ConfigMapEntry) error {
	log.Printf("Refreshing dashboard: %s", dashboard.Key)
	err := grafana.PushDashboard(dashboard.Value)
	if err != nil {
		configMapUpdates.WithLabelValues(dashboard.Name, dashboard.Namespace, "failed").Inc()
		return err
	}
	configMapUpdates.WithLabelValues(dashboard.Name, dashboard.Namespace, "success").Inc()
	return nil
}

func createConfigmapLW(kubeClient *kclient.Client) *kcache.ListWatch {
	return kcache.NewListWatchFromClient(kubeClient, "configmaps", kapi.NamespaceAll, kselector.Everything())
}

func watchForConfigmaps(kubeClient *kclient.Client, callback func(interface{})) kcache.Store {
	configmapStore, configmapController := kframework.NewInformer(
		createConfigmapLW(kubeClient),
		&kapi.ConfigMap{},
		0,
		kframework.ResourceEventHandlerFuncs{
			AddFunc:    callback,
			DeleteFunc: callback,
			UpdateFunc: func(a interface{}, b interface{}) { callback(b) },
		},
	)
	go configmapController.Run(wait.NeverStop)
	return configmapStore
}
