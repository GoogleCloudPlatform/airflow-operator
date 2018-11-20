/*
Copyright 2018 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"bytes"
	"fmt"
	resources "k8s.io/airflow-operator/pkg/controller/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math/rand"
	"strconv"
	"time"
)

// constants defining field values
const (
	ControllerVersion = "0.1"

	PasswordCharSpace = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	LifecycleManaged  = "managed"
	LifecycleReferred = "referred"

	ActionCheck  = "check"
	ActionCreate = "create"
	ActionDelete = "delete"

	LabelAirflowCR                 = "airflow-cr"
	ValueAirflowCRBase             = "airflow-base"
	ValueAirflowCRCluster          = "airflow-cluster"
	LabelAirflowCRName             = "airflow-cr-name"
	LabelAirflowComponent          = "airflow-component"
	ValueAirflowComponentMySQL     = "mysql"
	ValueAirflowComponentSQLProxy  = "sqlproxy"
	ValueAirflowComponentSQL       = "sql"
	ValueAirflowComponentUI        = "airflowui"
	ValueAirflowComponentNFS       = "nfs"
	ValueAirflowComponentRedis     = "redis"
	ValueAirflowComponentScheduler = "scheduler"
	ValueAirflowComponentWorker    = "worker"
	ValueAirflowComponentFlower    = "flower"
	LabelControllerVersion         = "airflow-controller-version"
	LabelApp                       = "app"

	KindAirflowBase    = "AirflowBase"
	KindAirflowCluster = "AirflowCluster"

	PodManagementPolicyParallel = "Parallel"

	GitSyncDestDir  = "gitdags"
	GCSSyncDestDir  = "dags"
	afk             = "AIRFLOW__KUBERNETES__"
	afc             = "AIRFLOW__CORE__"
	AirflowHome     = "/usr/local/airflow"
	AirflowDagsBase = AirflowHome + "/dags/"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// ResourceInfo is a container to capture the k8s resource info to be used by controller
type ResourceInfo struct {
	// Lifecycle can be: managed, reference
	Lifecycle string
	// Obj refers to the resource object  can be: sts, service, secret, pvc, ..
	Obj resources.ResourceHandle
	//Obj metav1.Object
	// Action - hint on action to be taken as part of Reconcile
	Action string
}

// ResourceSelector captures the k8s resource info and selector to fetch child resources
type ResourceSelector struct {
	// Obj refers to the resource object  can be: sts, service, secret, pvc, ..
	Obj resources.ResourceHandle
	// Selector - selector to pass to list
	Selectors labels.Selector
}

// ComponentHandle is an interface for operating on Resource
type ComponentHandle interface {
	ExpectedResources(rsrc interface{}) []ResourceInfo
	ObserveSelectors(rsrc interface{}) []ResourceSelector
	Differs(expected ResourceInfo, observed ResourceInfo) bool
	UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error)
}

// AirflowResource an interface for operating on AirflowResource
type AirflowResource interface {
	getMeta(name string, labels map[string]string) metav1.ObjectMeta
	getAnnotations() map[string]string
	getAffinity() *corev1.Affinity
	getNodeSelector() map[string]string
	getName() string
	getNameSpace() string
	getLabels() map[string]string
	getCRName() string
}

func optionsToString(options map[string]string, prefix string) string {
	var buf bytes.Buffer
	for k, v := range options {
		buf.WriteString(fmt.Sprintf("%s%s %s ", prefix, k, v))
	}
	return buf.String()
}

// RandomAlphanumericString generates a random password of some fixed length.
func RandomAlphanumericString(strlen int) []byte {
	result := make([]byte, strlen)
	for i := range result {
		result[i] = PasswordCharSpace[random.Intn(len(PasswordCharSpace))]
	}
	return result
}

func envFromSecret(name string, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Key: key,
		},
	}
}

// RsrcLabels return resource labels
func RsrcLabels(cr, name, component string) map[string]string {
	return map[string]string{
		LabelAirflowCR:        cr,
		LabelAirflowCRName:    name,
		LabelAirflowComponent: component,
	}
}

func selectorLabels(r AirflowResource, component string) labels.Selector {
	return labels.Set(RsrcLabels(r.getCRName(), r.getName(), component)).AsSelector()
}

func rsrcName(name string, component string, suffix string) string {
	return name + "-" + component + suffix
}

func nameAndLabels(r AirflowResource, component string, suffix string, needlabels bool) (string, map[string]string, map[string]string) {
	name := rsrcName(r.getName(), component, suffix)

	if !needlabels {
		return name, nil, nil
	}

	labels := r.getLabels()
	matchlabels := make(map[string]string)
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[LabelAirflowCR] = r.getCRName()
	labels[LabelAirflowCRName] = r.getName()
	labels[LabelApp] = name
	labels[LabelAirflowComponent] = component
	labels[LabelControllerVersion] = ControllerVersion
	matchlabels[LabelAirflowCR] = r.getCRName()
	matchlabels[LabelAirflowCRName] = r.getName()
	matchlabels[LabelAirflowComponent] = component
	matchlabels[LabelApp] = name

	return name, labels, matchlabels
}

func (r *AirflowBase) getName() string { return r.Name }

func (r *AirflowBase) getNameSpace() string { return r.Namespace }

func (r *AirflowBase) getAnnotations() map[string]string { return r.Spec.Annotations }

func (r *AirflowBase) getAffinity() *corev1.Affinity { return r.Spec.Affinity }

func (r *AirflowBase) getNodeSelector() map[string]string { return r.Spec.NodeSelector }

func (r *AirflowBase) getLabels() map[string]string { return r.Spec.Labels }

func (r *AirflowBase) getCRName() string { return ValueAirflowCRBase }

func (r *AirflowBase) getMeta(name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   r.Namespace,
		Annotations: r.Spec.Annotations,
		Labels:      labels,
		Name:        name,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(r, schema.GroupVersionKind{
				Group:   SchemeGroupVersion.Group,
				Version: SchemeGroupVersion.Version,
				Kind:    KindAirflowBase,
			}),
		},
	}
}

func (r *AirflowCluster) getName() string { return r.Name }

func (r *AirflowCluster) getNameSpace() string { return r.Namespace }

func (r *AirflowCluster) getAnnotations() map[string]string { return r.Spec.Annotations }

func (r *AirflowCluster) getAffinity() *corev1.Affinity { return r.Spec.Affinity }

func (r *AirflowCluster) getNodeSelector() map[string]string { return r.Spec.NodeSelector }

func (r *AirflowCluster) getLabels() map[string]string { return r.Spec.Labels }

func (r *AirflowCluster) getCRName() string { return ValueAirflowCRCluster }

func (r *AirflowCluster) getMeta(name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   r.Namespace,
		Annotations: r.Spec.Annotations,
		Labels:      labels,
		Name:        name,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(r, schema.GroupVersionKind{
				Group:   SchemeGroupVersion.Group,
				Version: SchemeGroupVersion.Version,
				Kind:    KindAirflowCluster,
			}),
		},
	}
}

func (r *AirflowCluster) getAirflowPrometheusEnv() []corev1.EnvVar {
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	ap := "AIRFLOW_PROMETHEUS_"
	apd := ap + "DATABASE_"
	env := []corev1.EnvVar{
		{Name: ap + "LISTEN_ADDR", Value: ":9112"},
		{Name: apd + "BACKEND", Value: "mysql"},
		{Name: apd + "HOST", Value: sqlSvcName},
		{Name: apd + "PORT", Value: "3306"},
		{Name: apd + "USER", Value: r.Spec.Scheduler.DBUser},
		{Name: apd + "PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: apd + "NAME", Value: r.Spec.Scheduler.DBName},
	}
	return env
}

func (r *AirflowCluster) getAirflowEnv(saName string) []corev1.EnvVar {
	sp := r.Spec
	sqlSvcName := rsrcName(sp.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	redisSecret, _, _ := nameAndLabels(r, ValueAirflowComponentRedis, "", false)
	redisSvcName := redisSecret
	dagFolder := AirflowDagsBase
	if sp.DAGs != nil {
		if sp.DAGs.Git != nil {
			dagFolder = AirflowDagsBase + "/" + GitSyncDestDir + "/" + sp.DAGs.DagSubdir
		} else if sp.DAGs.GCS != nil {
			dagFolder = AirflowDagsBase + "/" + GCSSyncDestDir + "/" + sp.DAGs.DagSubdir
		}
	}
	env := []corev1.EnvVar{
		{Name: "EXECUTOR", Value: sp.Executor},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: afc + "DAGS_FOLDER", Value: dagFolder},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "SQL_USER", Value: sp.Scheduler.DBUser},
		{Name: "SQL_DB", Value: sp.Scheduler.DBName},
		{Name: "DB_TYPE", Value: "mysql"},
	}
	if sp.Executor == ExecutorK8s {
		env = append(env, []corev1.EnvVar{
			{Name: afk + "WORKER_CONTAINER_REPOSITORY", Value: sp.Worker.Image},
			{Name: afk + "WORKER_CONTAINER_TAG", Value: sp.Worker.Version},
			{Name: afk + "WORKER_CONTAINER_IMAGE_PULL_POLICY", Value: "IfNotPresent"},
			{Name: afk + "DELETE_WORKER_PODS", Value: "True"},
			{Name: afk + "NAMESPACE", Value: r.Namespace},
			//{Name: afk+"AIRFLOW_CONFIGMAP", Value: ??},
			//{Name: afk+"IMAGE_PULL_SECRETS", Value: s.ImagePullSecrets},
			//{Name: afk+"GCP_SERVICE_ACCOUNT_KEYS", Vaslue:  ??},
		}...)
		if sp.DAGs != nil && sp.DAGs.Git != nil {
			env = append(env, []corev1.EnvVar{
				{Name: afk + "GIT_REPO", Value: sp.DAGs.Git.Repo},
				{Name: afk + "GIT_BRANCH", Value: sp.DAGs.Git.Branch},
				{Name: afk + "GIT_SUBPATH", Value: sp.DAGs.DagSubdir},
				{Name: afk + "WORKER_SERVICE_ACCOUNT_NAME", Value: saName},
			}...)
			if sp.DAGs.Git.CredSecretRef != nil {
				env = append(env, []corev1.EnvVar{
					{Name: "GIT_PASSWORD",
						ValueFrom: envFromSecret(sp.DAGs.Git.CredSecretRef.Name, "password")},
					{Name: "GIT_USER", Value: sp.DAGs.Git.User},
				}...)
			}
		}
	}
	if sp.Executor == ExecutorCelery {
		env = append(env,
			[]corev1.EnvVar{
				{Name: "REDIS_PASSWORD",
					ValueFrom: envFromSecret(redisSecret, "password")},
				{Name: "REDIS_HOST", Value: redisSvcName},
			}...)
	}
	return env
}

func (r *AirflowCluster) addAirflowContainers(ss *appsv1.StatefulSet, containers []corev1.Container, volName string) {
	ss.Spec.Template.Spec.InitContainers = []corev1.Container{}
	if r.Spec.DAGs != nil {
		init, dagContainer := r.Spec.DAGs.container(volName)
		if init {
			ss.Spec.Template.Spec.InitContainers = []corev1.Container{dagContainer}
		} else {
			containers = append(containers, dagContainer)
		}
	}
	ss.Spec.Template.Spec.Containers = containers
}

func (r *AirflowCluster) addMySQLUserDBContainer(ss *appsv1.StatefulSet) {
	sqlRootSecret := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	env := []corev1.EnvVar{
		{Name: "SQL_ROOT_PASSWORD", ValueFrom: envFromSecret(sqlRootSecret, "rootpassword")},
		{Name: "SQL_DB", Value: r.Spec.Scheduler.DBName},
		{Name: "SQL_USER", Value: r.Spec.Scheduler.DBUser},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "DB_TYPE", Value: "mysql"},
	}
	containers := []corev1.Container{
		{
			Name:    "mysql-dbcreate",
			Image:   defaultMySQLImage + ":" + defaultMySQLVersion,
			Env:     env,
			Command: []string{"/bin/bash"},
			//SET GLOBAL explicit_defaults_for_timestamp=ON;
			Args: []string{"-c", `
mysql -uroot -h$(SQL_HOST) -p$(SQL_ROOT_PASSWORD) << EOSQL 
CREATE DATABASE IF NOT EXISTS $(SQL_DB);
USE $(SQL_DB);
CREATE USER IF NOT EXISTS '$(SQL_USER)'@'%' IDENTIFIED BY '$(SQL_PASSWORD)';
GRANT ALL ON $(SQL_DB).* TO '$(SQL_USER)'@'%' ;
FLUSH PRIVILEGES;
EOSQL
`},
		},
	}
	ss.Spec.Template.Spec.InitContainers = append(containers, ss.Spec.Template.Spec.InitContainers...)
}

// sts returns a StatefulSet object which specifies
//  CPU and memory
//  resources
//  volume, volume mount
//  pod spec
func sts(r AirflowResource, component string, suffix string, svc bool) *appsv1.StatefulSet {
	name, labels, matchlabels := nameAndLabels(r, component, suffix, true)
	svcName := ""
	if svc {
		svcName = name
	}

	return &appsv1.StatefulSet{
		ObjectMeta: r.getMeta(name, labels),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: svcName,
			Selector: &metav1.LabelSelector{
				MatchLabels: matchlabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: r.getAnnotations(),
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					Affinity:     r.getAffinity(),
					NodeSelector: r.getNodeSelector(),
					Subdomain:    name,
				},
			},
		},
	}
}

func service(r AirflowResource, component string, name string, ports []corev1.ServicePort) *resources.Service {
	sname, labels, matchlabels := nameAndLabels(r, component, "", true)
	if name == "" {
		name = sname
	}
	return &resources.Service{
		Service: &corev1.Service{
			ObjectMeta: r.getMeta(name, labels),
			Spec: corev1.ServiceSpec{
				Ports:    ports,
				Selector: matchlabels,
			},
		},
	}
}

func podDisruption(r AirflowResource, component string, suffix string, minavail string) *resources.PodDisruptionBudget {
	name, label, selectors := nameAndLabels(r, component, suffix, true)
	minAvailable := intstr.Parse(minavail)

	return &resources.PodDisruptionBudget{
		PodDisruptionBudget: &policyv1.PodDisruptionBudget{
			ObjectMeta: r.getMeta(name, label),
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvailable,
				Selector: &metav1.LabelSelector{
					MatchLabels: selectors,
				},
			},
		},
	}
}

// ------------------------------ MYSQL  ---------------------------------------

func (s *MySQLSpec) service(r *AirflowBase) *resources.Service {
	return service(r, ValueAirflowComponentMySQL,
		rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		[]corev1.ServicePort{{Name: "mysql", Port: 3306}})
}

func (s *MySQLSpec) podDisruption(r *AirflowBase) *resources.PodDisruptionBudget {
	return podDisruption(r, ValueAirflowComponentMySQL, "", "100%")
}

func (s *MySQLSpec) secret(r *AirflowBase) *resources.Secret {
	name, labels, _ := nameAndLabels(r, ValueAirflowComponentSQL, "", true)
	return &resources.Secret{
		Secret: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password":     RandomAlphanumericString(16),
				"rootpassword": RandomAlphanumericString(16),
			},
		},
	}
}

func (s *MySQLSpec) sts(r *AirflowBase) *resources.StatefulSet {
	sqlSecret, _, _ := nameAndLabels(r, ValueAirflowComponentSQL, "", false)
	ss := sts(r, ValueAirflowComponentMySQL, "", true)
	ss.Spec.Replicas = &s.Replicas
	volName := "mysql-data"
	if s.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.VolumeClaimTemplate}
		volName = s.VolumeClaimTemplate.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "mysql",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "MYSQL_DATABASE", Value: "testdb"},
				{Name: "MYSQL_USER", Value: "airflow"},
				{Name: "MYSQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
				{Name: "MYSQL_ROOT_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "rootpassword")},
			},
			//Args:      []string{"-c", fmt.Sprintf("exec mysql %s", optionsToString(s.Options, "--"))},
			Args:      []string{"--explicit-defaults-for-timestamp=ON"},
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "mysql",
					ContainerPort: 3306,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/var/lib/mysql",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"bash", "-c", "mysqladmin -p$MYSQL_ROOT_PASSWORD ping"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       20,
				TimeoutSeconds:      5,
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"bash", "-c", "mysql -u$MYSQL_USER -p$MYSQL_PASSWORD -e \"use testdb\""},
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      2,
			},
		},
	}
	return &resources.StatefulSet{StatefulSet: ss}
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *MySQLSpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowBase)
	if s.Operator {
		return nil
	}
	rsrcInfos := []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.secret(r), ""},
		ResourceInfo{LifecycleManaged, s.service(r), ""},
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
		ResourceInfo{LifecycleManaged, s.podDisruption(r), ""},
	}
	//if s.VolumeClaimTemplate != nil {
	//	rsrcInfos = append(rsrcInfos, ResourceInfo{LifecycleReferred, s.VolumeClaimTemplate, ""})
	//}
	return rsrcInfos
}

// ObserveSelectors returns the list of resource/selecitos for those resources created by
// the operator for this spec and those resources referenced by this operator.
func (s *MySQLSpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowBase)
	if s.Operator {
		return nil
	}
	selector := selectorLabels(r, ValueAirflowComponentMySQL)
	secretSelector := selectorLabels(r, ValueAirflowComponentSQL)
	rsrcSelectos := []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.Service{}, selector},
		{&resources.Secret{}, secretSelector},
		{&resources.PodDisruptionBudget{}, selector},
	}
	//if s.VolumeClaimTemplate != nil {
	//	rsrcSelectos = append(rsrcSelectos, ResourceSelector{s.VolumeClaimTemplate, nil})
	//}
	return rsrcSelectos
}

// Differs returns true if the resource needs to be updated
func (s *MySQLSpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	switch expected.Obj.(type) {
	case *resources.Secret:
		// Dont update a secret
		return false
	case *resources.Service:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
		expected.Obj.(*resources.Service).Spec.ClusterIP = observed.Obj.(*resources.Service).Spec.ClusterIP
	case *resources.PodDisruptionBudget:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *MySQLSpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowBaseStatus)
	status.MySQL = ComponentStatus{}
	if s != nil {
		status.MySQL.update(reconciled, err)
		if status.MySQL.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

// ------------------------------ Airflow UI ---------------------------------------

func (s *AirflowUISpec) secret(r *AirflowCluster) *resources.Secret {
	name, labels, _ := nameAndLabels(r, ValueAirflowComponentUI, "", true)
	return &resources.Secret{
		Secret: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password": RandomAlphanumericString(16),
			},
		},
	}
}

// ExpectedResources returns the list of resource/name for those resources created by
func (s *AirflowUISpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowCluster)
	return []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
		ResourceInfo{LifecycleManaged, s.secret(r), ""},
	}
}

// ObserveSelectors returns the list of resource/selecitos for resources created
func (s *AirflowUISpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowCluster)
	selector := selectorLabels(r, ValueAirflowComponentUI)
	return []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.Secret{}, selector},
	}
}

// Differs returns true if the resource needs to be updated
func (s *AirflowUISpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	// TODO
	switch expected.Obj.(type) {
	case *resources.Secret:
		// Dont update a secret
		return false
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *AirflowUISpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowClusterStatus)
	status.UI = ComponentStatus{}
	if s != nil {
		status.UI.update(reconciled, err)
		if status.UI.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

func (s *AirflowUISpec) sts(r *AirflowCluster) *resources.StatefulSet {
	volName := "dags-data"
	ss := sts(r, ValueAirflowComponentUI, "", false)
	ss.Spec.Replicas = &s.Replicas
	ss.Spec.PodManagementPolicy = PodManagementPolicyParallel
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"webserver"}
	env := r.getAirflowEnv(ss.Name)
	containers := []corev1.Container{
		{
			//imagePullPolicy: "Always"
			//envFrom:
			Name:            "airflow-ui",
			Image:           s.Image + ":" + s.Version,
			Env:             env,
			ImagePullPolicy: corev1.PullAlways,
			Args:            args,
			Resources:       s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "web",
					ContainerPort: 8080,
					//Protocol:      "TCP",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/health",
						Port: intstr.FromString("web"),
					},
				},
				InitialDelaySeconds: 100,
				PeriodSeconds:       60,
				TimeoutSeconds:      2,
				SuccessThreshold:    1,
				FailureThreshold:    5,
			},
		},
	}

	r.addAirflowContainers(ss, containers, volName)
	r.addMySQLUserDBContainer(ss)
	return &resources.StatefulSet{StatefulSet: ss}
}

// ------------------------------ NFSStoreSpec ---------------------------------------

// ExpectedResources returns the list of resource/name for those resources created by
func (s *NFSStoreSpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowBase)
	return []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
		ResourceInfo{LifecycleManaged, s.service(r), ""},
		ResourceInfo{LifecycleManaged, s.podDisruption(r), ""},
	}
}

// ObserveSelectors returns the list of resource/selecitos for resources created
func (s *NFSStoreSpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowBase)
	selector := selectorLabels(r, ValueAirflowComponentNFS)
	return []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.Service{}, selector},
		{&resources.PodDisruptionBudget{}, selector},
	}
}

func (s *NFSStoreSpec) sts(r *AirflowBase) *resources.StatefulSet {
	ss := sts(r, ValueAirflowComponentNFS, "", true)
	ss.Spec.PodManagementPolicy = PodManagementPolicyParallel
	volName := "nfs-data"
	if s.Volume != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.Volume}
		volName = s.Volume.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			//imagePullPolicy: "Always"
			//envFrom:
			Name:      "nfs-server",
			Image:     s.Image + ":" + s.Version,
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{Name: "nfs", ContainerPort: 2049},
				{Name: "mountd", ContainerPort: 20048},
				{Name: "rpcbind", ContainerPort: 111},
			},
			SecurityContext: &corev1.SecurityContext{},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/exports",
				},
			},
		},
	}
	return &resources.StatefulSet{StatefulSet: ss}
}

func (s *NFSStoreSpec) podDisruption(r *AirflowBase) *resources.PodDisruptionBudget {
	return podDisruption(r, ValueAirflowComponentNFS, "", "100%")
}

func (s *NFSStoreSpec) service(r *AirflowBase) *resources.Service {
	return service(r, ValueAirflowComponentNFS, "",
		[]corev1.ServicePort{
			{Name: "nfs", Port: 2049},
			{Name: "mountd", Port: 20048},
			{Name: "rpcbind", Port: 111},
		})
}

// Differs returns true if the resource needs to be updated
func (s *NFSStoreSpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	// TODO
	switch expected.Obj.(type) {
	case *resources.Service:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
		expected.Obj.(*resources.Service).Spec.ClusterIP = observed.Obj.(*resources.Service).Spec.ClusterIP
	case *resources.PodDisruptionBudget:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *NFSStoreSpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowBaseStatus)
	status.Storage = ComponentStatus{}
	if s != nil {
		status.Storage.update(reconciled, err)
		if status.Storage.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

// ------------------------------ SQLProxy ---------------------------------------
func (s *SQLProxySpec) service(r *AirflowBase) *resources.Service {
	return service(r, ValueAirflowComponentSQLProxy,
		rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		[]corev1.ServicePort{{Name: "sqlproxy", Port: 3306}})
}

func (s *SQLProxySpec) sts(r *AirflowBase) *resources.StatefulSet {
	ss := sts(r, ValueAirflowComponentSQLProxy, "", true)
	instance := s.Project + ":" + s.Region + ":" + s.Instance + "=tcp:0.0.0.0:3306"
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "sqlproxy",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "SQL_INSTANCE", Value: instance},
			},
			Command: []string{"/cloud_sql_proxy", "-instances", "$(SQL_INSTANCE)"},
			//volumeMounts:
			//- name: ssl-certs
			//mountPath: /etc/ssl/certs
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "sqlproxy",
					ContainerPort: 3306,
				},
			},
		},
	}
	return &resources.StatefulSet{StatefulSet: ss}
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *SQLProxySpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowBase)
	name, _, _ := nameAndLabels(r, ValueAirflowComponentSQL, "", false)
	return []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.service(r), ""},
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
		ResourceInfo{LifecycleReferred,
			&resources.Secret{
				Secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: r.Namespace,
						Name:      name,
					}}},
			""},
	}
}

// ObserveSelectors returns the list of resource/selecitos for those resources created by
// the operator for this spec and those resources referenced by this operator.
func (s *SQLProxySpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowBase)
	selector := selectorLabels(r, ValueAirflowComponentSQLProxy)
	svcselector := selectorLabels(r, ValueAirflowComponentSQLProxy)
	name, _, _ := nameAndLabels(r, ValueAirflowComponentSQL, "", false)
	return []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.Service{}, svcselector},
		{&resources.Secret{
			Secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: r.Namespace,
					Name:      name,
				}}},
			nil},
	}
}

// Differs returns true if the resource needs to be updated
func (s *SQLProxySpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	switch expected.Obj.(type) {
	case *resources.Service:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
		expected.Obj.(*resources.Service).Spec.ClusterIP = observed.Obj.(*resources.Service).Spec.ClusterIP
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *SQLProxySpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowBaseStatus)
	status.SQLProxy = ComponentStatus{}
	if s != nil {
		status.SQLProxy.update(reconciled, err)
		if status.SQLProxy.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

// ------------------------------ RedisSpec ---------------------------------------
func (s *RedisSpec) service(r *AirflowCluster) *resources.Service {
	return service(r, ValueAirflowComponentRedis, "",
		[]corev1.ServicePort{{Name: "redis", Port: 6379}})
}

func (s *RedisSpec) podDisruption(r *AirflowCluster) *resources.PodDisruptionBudget {
	return podDisruption(r, ValueAirflowComponentRedis, "", "100%")
}

func (s *RedisSpec) secret(r *AirflowCluster) *resources.Secret {
	name, labels, _ := nameAndLabels(r, ValueAirflowComponentRedis, "", true)
	return &resources.Secret{
		Secret: &corev1.Secret{
			ObjectMeta: r.getMeta(name, labels),
			Data: map[string][]byte{
				"password": RandomAlphanumericString(16),
			},
		},
	}
}

func (s *RedisSpec) sts(r *AirflowCluster) *resources.StatefulSet {
	redisSecret, _, _ := nameAndLabels(r, ValueAirflowComponentRedis, "", false)
	ss := sts(r, ValueAirflowComponentRedis, "", true)
	volName := "redis-data"
	if s.VolumeClaimTemplate != nil {
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*s.VolumeClaimTemplate}
		volName = s.VolumeClaimTemplate.Name
	} else {
		ss.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
	}
	args := []string{"--requirepass", "$(REDIS_PASSWORD)"}
	if s.AdditionalArgs != "" {
		args = append(args, "$(REDIS_EXTRA_FLAGS)")
	}
	ss.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  "redis",
			Image: s.Image + ":" + s.Version,
			Env: []corev1.EnvVar{
				{Name: "REDIS_EXTRA_FLAGS", Value: s.AdditionalArgs},
				{Name: "REDIS_PASSWORD", ValueFrom: envFromSecret(redisSecret, "password")},
			},
			Args:      args,
			Resources: s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "redis",
					ContainerPort: 6379,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/data",
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"redis-cli", "ping"},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       20,
				TimeoutSeconds:      5,
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"redis-cli", "ping"},
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      2,
			},
		},
	}
	return &resources.StatefulSet{StatefulSet: ss}
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *RedisSpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowCluster)
	rsrcInfos := []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.secret(r), ""},
		ResourceInfo{LifecycleManaged, s.service(r), ""},
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
		ResourceInfo{LifecycleManaged, s.podDisruption(r), ""},
	}
	//if s.VolumeClaimTemplate != nil {
	//		rsrcInfos = append(rsrcInfos, ResourceInfo{LifecycleReferred, s.VolumeClaimTemplate, ""})
	//	}
	return rsrcInfos
}

// ObserveSelectors returns the list of resource/selecitos for those resources created by
// the operator for this spec and those resources referenced by this operator.
func (s *RedisSpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowCluster)
	selector := selectorLabels(r, ValueAirflowComponentRedis)
	rsrcSelectos := []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.Service{}, selector},
		{&resources.Secret{}, selector},
		{&resources.PodDisruptionBudget{}, selector},
	}
	//if s.VolumeClaimTemplate != nil {
	//	rsrcSelectos = append(rsrcSelectos, ResourceSelector{s.VolumeClaimTemplate, nil})
	//}
	return rsrcSelectos
}

// Differs returns true if the resource needs to be updated
func (s *RedisSpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	switch expected.Obj.(type) {
	case *resources.Secret:
		// Dont update a secret
		return false
	case *resources.Service:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
		expected.Obj.(*resources.Service).Spec.ClusterIP = observed.Obj.(*resources.Service).Spec.ClusterIP
	case *resources.PodDisruptionBudget:
		expected.Obj.SetResourceVersion(observed.Obj.GetResourceVersion())
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *RedisSpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowClusterStatus)
	status.Redis = ComponentStatus{}
	if s != nil {
		status.Redis.update(reconciled, err)
		if status.Redis.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

// ------------------------------ Scheduler ---------------------------------------

func (s *GCSSpec) container(volName string) (bool, corev1.Container) {
	init := false
	container := corev1.Container{}
	env := []corev1.EnvVar{
		{Name: "GCS_BUCKET", Value: s.Bucket},
	}
	if s.Once {
		init = true
	}
	container = corev1.Container{
		Name:  "gcs-syncd",
		Image: gcssyncImage + ":" + gcssyncVersion,
		Env:   env,
		Args:  []string{"/home/airflow/gcs"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volName,
				MountPath: "/home/airflow/gcs",
			},
		},
	}

	return init, container
}
func (s *GitSpec) container(volName string) (bool, corev1.Container) {
	init := false
	container := corev1.Container{}
	env := []corev1.EnvVar{
		{Name: "GIT_SYNC_REPO", Value: s.Repo},
		{Name: "GIT_SYNC_DEST", Value: GitSyncDestDir},
		{Name: "GIT_SYNC_BRANCH", Value: s.Branch},
		{Name: "GIT_SYNC_ONE_TIME", Value: strconv.FormatBool(s.Once)},
		{Name: "GIT_SYNC_REV", Value: s.Rev},
	}
	if s.CredSecretRef != nil {
		env = append(env, []corev1.EnvVar{
			{Name: "GIT_PASSWORD",
				ValueFrom: envFromSecret(s.CredSecretRef.Name, "password")},
			{Name: "GIT_USER", Value: s.User},
		}...)
	}
	if s.Once {
		init = true
	}
	container = corev1.Container{
		Name:    "git-sync",
		Image:   gitsyncImage + ":" + gitsyncVersion,
		Env:     env,
		Command: []string{"/git-sync"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "gitsync",
				ContainerPort: 2020,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volName,
				MountPath: "/git",
			},
		},
	}

	return init, container
}

func (s *DagSpec) container(volName string) (bool, corev1.Container) {
	init := false
	container := corev1.Container{}

	if s.Git != nil {
		return s.Git.container(volName)
	}
	if s.GCS != nil {
		return s.GCS.container(volName)
	}

	return init, container
}

func (s *SchedulerSpec) serviceaccount(r *AirflowCluster) *resources.ServiceAccount {
	name, labels, _ := nameAndLabels(r, ValueAirflowComponentScheduler, "", true)
	return &resources.ServiceAccount{
		ServiceAccount: &corev1.ServiceAccount{
			ObjectMeta: r.getMeta(name, labels),
		}}
}

func (s *SchedulerSpec) rb(r *AirflowCluster) *resources.RoleBinding {
	name, labels, _ := nameAndLabels(r, ValueAirflowComponentScheduler, "", true)
	return &resources.RoleBinding{
		RoleBinding: &rbacv1.RoleBinding{
			ObjectMeta: r.getMeta(name, labels),
			Subjects: []rbacv1.Subject{
				{Kind: "ServiceAccount", Name: name, Namespace: r.Namespace},
			},
			RoleRef: rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cluster-admin"},
		},
	}
}

func (s *SchedulerSpec) sts(r *AirflowCluster) *resources.StatefulSet {
	volName := "dags-data"
	ss := sts(r, ValueAirflowComponentScheduler, "", true)
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"scheduler"}

	if r.Spec.Executor == ExecutorK8s {
		ss.Spec.Template.Spec.ServiceAccountName = ss.Name
	}
	containers := []corev1.Container{
		{
			Name:            "scheduler",
			Image:           s.Image + ":" + s.Version,
			Env:             r.getAirflowEnv(ss.Name),
			ImagePullPolicy: corev1.PullAlways,
			Args:            args,
			Resources:       s.Resources,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
		},
		{
			Name:  "metrics",
			Image: "pbweb/airflow-prometheus-exporter:latest",
			Env:   r.getAirflowPrometheusEnv(),
			Ports: []corev1.ContainerPort{
				{
					Name:          "metrics",
					ContainerPort: 9112,
				},
			},
		},
	}
	r.addAirflowContainers(ss, containers, volName)
	return &resources.StatefulSet{StatefulSet: ss}
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *SchedulerSpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowCluster)
	rsrcInfos := []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.serviceaccount(r), ""},
		ResourceInfo{LifecycleManaged, s.rb(r), ""},
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
	}

	if r.Spec.DAGs != nil {
		git := r.Spec.DAGs.Git
		if git != nil && git.CredSecretRef != nil {
			rsrcInfos = append(rsrcInfos,
				ResourceInfo{LifecycleReferred,
					&resources.Secret{
						Secret: &corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: r.Namespace,
								Name:      git.CredSecretRef.Name,
							}}},
					""})
		}
	}

	return rsrcInfos
}

// ObserveSelectors returns the list of resource/selecitos for those resources created by
// the operator for this spec and those resources referenced by this operator.
func (s *SchedulerSpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowCluster)
	selector := selectorLabels(r, ValueAirflowComponentScheduler)
	rsrcSelectors := []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.ServiceAccount{}, selector},
		{&resources.RoleBinding{}, selector},
	}

	if r.Spec.DAGs != nil {
		git := r.Spec.DAGs.Git
		if git != nil && git.CredSecretRef != nil {
			rsrcSelectors = append(rsrcSelectors, ResourceSelector{
				&resources.Secret{
					Secret: &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: r.Namespace,
							Name:      git.CredSecretRef.Name,
						}}},
				nil})
		}
	}

	return rsrcSelectors
}

// Differs returns true if the resource needs to be updated
func (s *SchedulerSpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	switch expected.Obj.(type) {
	case *resources.ServiceAccount:
		// Dont update a SA
		return false
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *SchedulerSpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowClusterStatus)
	status.Scheduler = SchedulerStatus{}
	if s != nil {
		status.Scheduler.Resources.update(reconciled, err)
		if status.Scheduler.Resources.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

// ------------------------------ Worker -
func (s *WorkerSpec) sts(r *AirflowCluster) *resources.StatefulSet {
	ss := sts(r, ValueAirflowComponentWorker, "", true)
	volName := "dags-data"
	ss.Spec.Replicas = &s.Replicas
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"worker"}
	env := r.getAirflowEnv(ss.Name)
	containers := []corev1.Container{
		{
			Name:            "worker",
			Image:           s.Image + ":" + s.Version,
			Args:            args,
			Env:             env,
			ImagePullPolicy: corev1.PullAlways,
			Resources:       s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "wlog",
					ContainerPort: 8793,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
		},
	}
	r.addAirflowContainers(ss, containers, volName)
	return &resources.StatefulSet{StatefulSet: ss}
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *WorkerSpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowCluster)
	rsrcInfos := []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
	}
	// TODO storage spec ?
	return rsrcInfos
}

// ObserveSelectors returns the list of resource/selecitos for those resources created by
// the operator for this spec and those resources referenced by this operator.
func (s *WorkerSpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowCluster)
	selector := selectorLabels(r, ValueAirflowComponentWorker)
	rsrcSelectors := []ResourceSelector{
		{&resources.StatefulSet{}, selector},
	}
	return rsrcSelectors
}

// UpdateStatus use reconciled objects to update component status
func (s *WorkerSpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowClusterStatus)
	status.Worker = ComponentStatus{}
	if s != nil {
		status.Worker.update(reconciled, err)
		if status.Worker.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

// Differs returns true if the resource needs to be updated
func (s *WorkerSpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	// TODO
	return true
}

// ------------------------------ Flower ---------------------------------------

// ExpectedResources returns the list of resource/name for those resources created by
func (s *FlowerSpec) ExpectedResources(rsrc interface{}) []ResourceInfo {
	r := rsrc.(*AirflowCluster)
	return []ResourceInfo{
		ResourceInfo{LifecycleManaged, s.sts(r), ""},
	}
}

// ObserveSelectors returns the list of resource/selecitos for resources created
func (s *FlowerSpec) ObserveSelectors(rsrc interface{}) []ResourceSelector {
	r := rsrc.(*AirflowCluster)
	selector := selectorLabels(r, ValueAirflowComponentFlower)
	return []ResourceSelector{
		{&resources.StatefulSet{}, selector},
		{&resources.Secret{}, selector},
	}
}

// Differs returns true if the resource needs to be updated
func (s *FlowerSpec) Differs(expected ResourceInfo, observed ResourceInfo) bool {
	// TODO
	switch expected.Obj.(type) {
	case *resources.Secret:
		// Dont update a secret
		return false
	}
	return true
}

// UpdateStatus use reconciled objects to update component status
func (s *FlowerSpec) UpdateStatus(rsrc interface{}, reconciled []ResourceInfo, err error) {
	status := rsrc.(*AirflowClusterStatus)
	status.Flower = ComponentStatus{}
	if s != nil {
		status.Flower.update(reconciled, err)
		if status.Flower.Status != StatusReady {
			status.Status = StatusInProgress
		}
	}
}

func (s *FlowerSpec) sts(r *AirflowCluster) *resources.StatefulSet {
	ss := sts(r, ValueAirflowComponentFlower, "", true)
	volName := "dags-data"
	ss.Spec.Replicas = &s.Replicas
	ss.Spec.Template.Spec.Volumes = []corev1.Volume{
		{Name: volName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	args := []string{"flower"}
	env := r.getAirflowEnv(ss.Name)
	containers := []corev1.Container{
		{
			Name:            "flower",
			Image:           s.Image + ":" + s.Version,
			Args:            args,
			Env:             env,
			ImagePullPolicy: corev1.PullAlways,
			Resources:       s.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "flower",
					ContainerPort: 5555,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volName,
					MountPath: "/usr/local/airflow/dags/",
				},
			},
		},
	}
	r.addAirflowContainers(ss, containers, volName)
	return &resources.StatefulSet{StatefulSet: ss}
}
