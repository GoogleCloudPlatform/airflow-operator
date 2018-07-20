**This is not an officially supported Google product.**

# Airflow Operator

The Airflow Operator is a Kubernetes [operator](https://coreos.com/blog/introducing-operators.html) that manages the life cycle of an Airflow Deployment. An airflow deployment is split into 2 parts Base and Cluster represented my AirflowBase and AirflowCluster [custom resources](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/). 

The controller performs these jobs:
* Creates and manages the necessary Kubernetes resources for an Airflow deployment.
* Updates the corresponding Kubernetes resources when the Base or Cluster specification changes.
* Restores managed Kubernetes resources that are deleted.
* Supports creation of different kind of Airflow schedulers
* Supports mulitple AirflowClusters configuration per AirflowBase

## Project Status

The Airflow Operator is still under active development and has not been extensively tested in production environment. Backward compatibility of the APIs is not guaranteed for alpha releases.

## Prerequisites

* Version >= 1.9 of Kubernetes.
* Version 4.0.x of Redis
* Version 1.9 of Airflow (1.10.0rc2+ for k8s executor)
* Version 5.7 of MySQL


## Quick Start
#### Assumptions
You have a GKE cluster and kubeconfig points to that cluster.

Due to a [known issue](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#defining_permissions_in_a_role) 
in GKE, you will need to first grant yourself cluster-admin privileges before you can create custom roles and role 
bindings on a GKE cluster versioned 1.6 and up.

#### Airflow Operator installation
Installing the airflow operator creates the 'airflow-system' namespace and creates stateful set in that namespace for the operator.

```bash
# deploy airflow-controller
$ kubectl create -f manifests/install_deps.yaml
$ sleep 60 # wait for cluster role binding to propagate
$ kubectl create -f manifests/install_controller.yaml

# check airflow controller pods
$ kubectl get pod airflow-controller-manager-0 -n airflow-system

# get airflow controller logs
$ kubectl logs -f airflow-controller-manager-0 -n airflow-system
```

#### Running the Samples

The `hack/sample/` directory contains sample Airflow CRs

##### Running MySQL based samples

```bash
# deploy base components first
$ kubectl apply -f hack/sample/mysql-celery/base.yaml

# after 30-60s deploy cluster components 
# for celery + git as DAG source
$ kubectl apply -f hack/sample/mysql-celery/cluster.yaml
# for celery + gcs as DAG source (you need to update to point to your gcs bucket)
$ kubectl apply -f hack/sample/mysql-celery-gcs/cluster.yaml

# port forward to access the UI
$ kubectl port-forward mc-cluster-airflowui-0 8080:8080
#  OR
$ kubectl port-forward mcg-cluster-airflowui-0 8080:8080

# get status of the CRs
$ kubectl get airflowbase/mc-base -o yaml 
$ kubectl get airflowcluster/mc-cluster -o yaml 
#  OR
$ kubectl get airflowcluster/mcg-cluster -o yaml 


# we could deploy k8s based cluster against the same base components
$ kubectl apply -f hack/sample/mysql-k8s/cluster.yaml

# port forward to access the k8s executor base clyster UI (on port 8081)
$ kubectl port-forward mk-cluster-airflowui-0 8081:8080

# get status of the CRs
$ kubectl get airflowcluster/mk-cluster -o yaml 
```

##### Running CloudSQL based samples
CloudSQL(mysql)  needs to be setup on your project.
A root password needs to be created for the CloudSQL.
The information about the project, region, instance needs to be updated in hack/samples/cloudsql-celery/base.yaml.
A secret containing the root password as "rootpassword" needs to be created with the name "cc-base-sql" (base.name + "-sql"). Update the hack/sample/cloudsql-celery/sqlproxy-secret.yaml

```bash
# create secret
$ kubectl apply -f hack/sample/cloudsql-celery/sqlproxy-secret.yaml
# deploy base components first
$ kubectl apply -f hack/sample/cloudsql-celery/base.yaml
# after 30-60s deploy cluster components
$ kubectl apply -f hack/sample/cloudsql-celery/cluster.yaml
# port forward to access the UI (port 8082)
$ kubectl port-forward cc-cluster-airflowui-0 8082:8080
# get status of the CRs
$ kubectl get airflowbase/cc-base -o yaml 
$ kubectl get airflowcluster/cc-cluster -o yaml 
```

## Supporting Kubernetes Resources

#### StatefulSet

- For MySQL server
- For Redis server
- For NFS server
- For Airflow-UI server
- For Airflow-Scheduler
- For Airflow-Workers

#### Secret

- Generated for MySQL user and root passwords
- Generated for Redis password

#### Service

- client service for MySQL
- client service for Redis
- client service for Airflow UI
- client service for NFS

#### PodDisruptionBudget

- for MySQL
- for Redis
- for NFS

#### RoleBinding

- for Scheduler

#### ServiceAccount

- for Scheduler

## Development
Local build and testing
```bash
# from repo top level folder
# uses current cluster from ~/.kube/config

# build
make build

# test
make test

# run locally
make run

# build docker image
# note: change the 'image' in makefile
make image

# push docker image
make push
```
