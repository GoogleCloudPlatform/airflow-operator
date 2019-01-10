[![Go Report Card](https://goreportcard.com/badge/github.com/GoogleCloudPlatform/airflow-operator)](https://goreportcard.com/report/github.com/GoogleCloudPlatform/airflow-operator)

**This is not an officially supported Google product.**

## Community

* Join our [Slack channel](https://kubernetes.slack.com/messages/CC1UAMYSV).

## Project Status

*Alpha*

The Airflow Operator is still under active development and has not been extensively tested in production environment. Backward compatibility of the APIs is not guaranteed for alpha releases.

## Prerequisites
* Version >= 1.9 of Kubernetes.
* Uses 1.9 of Airflow (1.10.1+ for k8s executor)
* Uses 4.0.x of Redis (for celery operator)
* Uses 5.7 of MySQL

## Get Started

[One Click Deployment](https://console.cloud.google.com/marketplace/details/google/airflow-operator) from Google Cloud Marketplace to your [GKE cluster](https://cloud.google.com/kubernetes-engine/)

Get started quickly with the Airflow Operator using the [Quick Start Guide](https://github.com/GoogleCloudPlatform/airflow-operator/blob/master/docs/quickstart.md)

For more information check the [Design](https://github.com/GoogleCloudPlatform/airflow-operator/blob/master/docs/design.md) and detailed [User Guide](https://github.com/GoogleCloudPlatform/airflow-operator/blob/master/docs/userguide.md)

## Airflow Operator Overview
Airflow Operator is a custom [Kubernetes operator](https://coreos.com/blog/introducing-operators.html) that makes it easy to deploy and manage [Apache Airflow](https://airflow.apache.org/) on Kubernetes. Apache Airflow is a platform to programmatically author, schedule and monitor workflows. Using the Airflow Operator, an Airflow cluster is split into 2 parts represented by the `AirflowBase` and `AirflowCluster` custom resources.
The Airflow Operator performs these jobs:
* Creates and manages the necessary Kubernetes resources for an Airflow deployment.
* Updates the corresponding Kubernetes resources when the `AirflowBase` or `AirflowCluster` specification changes.
* Restores managed Kubernetes resources that are deleted.
* Supports creation of Airflow schedulers with different Executors
* Supports sharing of the `AirflowBase` across mulitple `AirflowClusters`

Checkout out the [Design](https://github.com/GoogleCloudPlatform/airflow-operator/blob/master/docs/design.md)

![Airflow Cluster](docs/airflow-cluster.png)


## Development

Refer to the [Design](https://github.com/GoogleCloudPlatform/airflow-operator/blob/master/docs/design.md) and [Development Guide](https://github.com/GoogleCloudPlatform/airflow-operator/blob/master/docs/development.md).

## Managed Airflow solution

[Google Cloud Composer](https://cloud.google.com/composer/) is a fully managed workflow orchestration service targeting customers that need a workflow manager in the cloud.
