# Orbs Troubleshooting Guide

[Back to README](README.md)

## Production troubleshooting

### Dashboard

Raw metrics for a specific node, for a specific virtual chain are available: 

* `http:///metrics.json` endpoint

  > For example: [http://validator.orbs.com/vchains/1100000/metrics](http://validator.orbs.com/vchains/1100000/metrics)
  
* Collecting Prometheus metrics using `/metrics.prometheus` endpoint

* Visualizing Prometheus metrics with Grafana

Presently dashboards are not public, we plan to fix this.

  > [Orbs DevOps](https://orbsnetwork.grafana.net/d/Eqvddt3iz/orbs-devops?orgId=1&from=now-3h&to=now&refresh=15s) dashboard

### Logs

We ship error logs to logz.io, where they can be analyzed.
Presently they are not public, we are considering whether public access is necessary.

  > [logz.io](https://app.logz.io/#/dashboard/kibana/discover/4501ce90-4638-11e9-b5c5-c306d6d38229?_g=()) 