#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
image = gcr.io/airflow-operator/airflow-operator:v1alpha1
curdir = $(shell pwd)

all:
	echo "usage: make build|run|test|docker-build|docker-push|install|uninstall|install-cr|uninstall-cr"
build:
	GOBIN=$(curdir)/bin go install ./cmd/controller-manager
run: build
	bin/controller-manager  --kubeconfig ~/.kube/config
test: build
	TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd go test ./pkg/... ./cmd/...
docker-build: build
	docker build . -t $(image) -f docker/Dockerfile.controller
docker-push: docker-build
	docker push $(image)
install:
	kubectl create -f manifests/install_deps.yaml
	sleep 30
	kubectl create -f manifests/install_controller.yaml
uninstall:
	kubectl delete -f manifests/install_controller.yaml
	sleep 10
	kubectl delete -f manifests/install_deps.yaml
install-cr:
	kubectl create -f hack/sample/mysql-celery/base.yaml || true
	sleep 45
	kubectl create -f hack/sample/mysql-celery/cluster.yaml || true
uninstall-cr:
	kubectl delete -f hack/sample/mysql-celery/cluster.yaml || true
	sleep 10
	kubectl delete -f hack/sample/mysql-celery/base.yaml || true
