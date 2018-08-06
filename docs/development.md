# Development
You should have kubeconfig setup to point to your cluster.
In case you want to build the Airflow Operator from the source code, e.g., to test a fix or a feature you write, you can do so following the instructions below.

Cloning the repo:
```bash
$ mkdir -p $GOPATH/src/k8s.io
$ cd $GOPATH/src/k8s.io
$ git clone git@github.com:GoogleCloudPlatform/airflow-operator.git
```

Building and running locally:
```bash
# build
make build

# test
make test

# run locally
make run
```

To build a docker image and work with that, you would need to change the `image` in `Makefile` and `manifests/install_controller.yaml`
```bash
# building docker image
# note: change the 'image' in makefile
make docker-build

# push docker image
make docker-push
```
