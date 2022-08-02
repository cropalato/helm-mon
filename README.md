# helm-mon

This project will monitor all helm charts installed in your cluster and expose as prometheus metrics when a chart is overdue.

## Example how prometheus metrics looks like

```
# HELP helm_chart_overdue The number of versions the chart is overdue.
# TYPE helm_chart_overdue gauge
helm_chart_overdue{chart="cert-manager",namespace="cert-manager",version="v1.5.3"} 0
helm_chart_overdue{chart="ingress-nginx",namespace="ingress-nginx",version="3.34.0"} -1
helm_chart_overdue{chart="vault",namespace="vault",version="0.15.0"} 1
```

where:
| value | description                                     |
|-------|-------------------------------------------------|
|  -1   | The chart has not been found in monitored repos |
|   0   | The chart is up-to-date                         |
|  >1   | Number of versions the chart is overdue.        |


## How to test locally

This project requires a kube config to access the cluster kubernetes and a helm file with all repose you want to use to check the upgrades.

```bash
docker run --rm --name helm-mon -v ${HOME}/.kube:/root/.kube -v ${HOME}/.config/helm:/root/.config/helm -p 2112:2112 cropalato/helm-mon:v0.1.0
```


