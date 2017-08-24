# Kubernetes watcher for Grafana Dashboards and DataSources

[![Docker Build Statu](https://img.shields.io/docker/build/pierrevincent/k8s-grafana-watcher.svg)]() [![Docker Build Statu](https://img.shields.io/docker/pulls/pierrevincent/k8s-grafana-watcher.svg)]()

Enables automatic detection and update of Grafana dashboards and Datasources contained in Kubernetes configmaps.

By running this watcher in the same pod as Grafana, dashboards and datasources can be discovered as Grafana starts and while it's running. This has the benefit of removing the need for persistent storage, while ensuring that a re-scheduled pod will reload configured dashboards.

The typical workflow here is that the Grafana dashboards are treated as code and should be pushed through configmaps, rather than manually modified in Grafana itself.

## Docker image

Latest version of this image can be pulled from [Docker Hub](https://hub.docker.com/r/pierrevincent/k8s-grafana-watcher):

```
pierrevincent/k8s-grafana-watcher
```

## Parameters

- `-dashboardAnnotation` (defaults to env `CONFIG_MAP_DASHBOARD_ANNOTATION`): Annotation to indicate configmaps containing Grafana Dashboards
- `-datasourceAnnotation` (defaults to env `CONFIG_MAP_DATASOURCE_ANNOTATION`): Annotation to indicate configmaps containing Grafana Datasource
- `-grafanaUrl` (defaults to env `GRAFANA_URL`): Grafana URL to POST updates
- `-grafanaUsername` (defaults to env `GRAFANA_USERNAME`): Username for Grafana API authentication
- `-grafanaPassword` (defaults to env `GRAFANA_PASSWORD`): Password for Grafana API authentication

## Dashboard and Datasources configmaps

Examples of config maps for Dashboards and Datasources can be found in [example](example) directory.

Note that dashboard JSON should specify `"id": null`, to allow both creation and update.

## Deployment

This watcher is best run in the same pod as Grafana (you can run it outside too, but it is built with the life-cycle of the Grafana container in mind).

See [deployment](deployment) for an example of a Kubernetes deployment & service combining Grafana and the grafana-watcher.

## Work in Progress

Note that this watcher is a rough proof-of-concept, please be aware of the following if you decide to make use of it:

- Datasources can only be created at the moment
- Removing configmaps (or entries in configmaps) does not remove the corresponding Dashboards or Datasources
- Renaming Dashboards and Datasources won't remove the old ones

## Contributions welcome

Feel free to create issues or pull requests if you see anything that can be improved.

## Credits

This watcher was inspired by the work done on the [Prometheus Rule Loader](https://github.com/nordstrom/prometheusRuleLoader), kudos on the idea!
