/*
Copyright 2018 Google LLC.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/kubesdk/pkg/component"
	cr "sigs.k8s.io/kubesdk/pkg/customresource"
	"sigs.k8s.io/kubesdk/pkg/status"
)

// defaults and constant strings
const (
	defaultRedisImage       = "redis"
	defaultRedisVersion     = "4.0"
	defaultWorkerImage      = "gcr.io/airflow-operator/airflow"
	defaultSchedulerImage   = "gcr.io/airflow-operator/airflow"
	defaultFlowerImage      = "gcr.io/airflow-operator/airflow"
	gitsyncImage            = "gcr.io/google_containers/git-sync"
	gitsyncVersion          = "v3.0.1"
	gcssyncImage            = "gcr.io/cloud-airflow-releaser/gcs-syncd"
	gcssyncVersion          = "cloud_composer_service_2018-05-23-RC0"
	ExecutorLocal           = "Local"
	ExecutorCelery          = "Celery"
	ExecutorSequential      = "Sequential"
	ExecutorK8s             = "Kubernetes"
	defaultExecutor         = ExecutorLocal
	defaultBranch           = "master"
	defaultWorkerVersion    = "1.10.1"
	defaultSchedulerVersion = "1.10.1"
)

var allowedExecutors = []string{ExecutorLocal, ExecutorSequential, ExecutorCelery, ExecutorK8s}

// RedisSpec defines the attributes and desired state of Redis component
type RedisSpec struct {
	// Image defines the Redis Docker image name
	// +optional
	Image string `json:"image,omitempty"`
	// Version defines the Redis Docker image version.
	// +optional
	Version string `json:"version,omitempty"`
	// Flag when True generates RedisReplica CustomResource to be handled by Redis Operator
	// If False, a StatefulSet with 1 replica is created
	// +optional
	Operator bool `json:"operator,omitempty"`
	// Resources is the resource requests and limits for the pods.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// VolumeClaimTemplate allows a user to specify volume claim for MySQL Server files
	// +optional
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
	// AdditionalArgs for redis-server
	// +optional
	AdditionalArgs string `json:"additionalargs,omitempty"`
}

func (s *RedisSpec) validate(fp *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if s == nil {
		return errs
	}
	if s.Operator == true {
		errs = append(errs, field.Invalid(fp.Child("operator"), "", "Operator is not supported in this version"))
	}
	return errs
}

// FlowerSpec defines the attributes to deploy Flower component
type FlowerSpec struct {
	// Image defines the Flower Docker image.
	// +optional
	Image string `json:"image,omitempty"`
	// Version defines the Flower Docker image version.
	// +optional
	Version string `json:"version,omitempty"`
	// Replicas defines the number of running Flower instances in a cluster
	Replicas int32 `json:"replicas,omitempty"`
	// Resources is the resource requests and limits for the pods.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

func (s *FlowerSpec) validate(fp *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	return errs
}

// SchedulerSpec defines the attributes and desired state of Airflow Scheduler
type SchedulerSpec struct {
	// Image defines the Airflow custom server Docker image.
	// +optional
	Image string `json:"image,omitempty"`
	// Version defines the Airflow Docker image version
	// +optional
	Version string `json:"version,omitempty"`
	// DBName defines the Airflow Database to be used
	// +optional
	DBName string `json:"database,omitempty"`
	// DBUser defines the Airflow Database user to be used
	// +optional
	DBUser string `json:"dbuser,omitempty"`
	// Resources is the resource requests and limits for the pods.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

func (s *SchedulerSpec) validate(fp *field.Path) field.ErrorList {
	return field.ErrorList{}
}

// WorkerSpec defines the attributes and desired state of Airflow workers
type WorkerSpec struct {
	// Image defines the Airflow worker Docker image.
	// +optional
	Image string `json:"image,omitempty"`
	// Version defines the Airflow worker Docker image version
	// +optional
	Version string `json:"version,omitempty"`
	// Replicas is the count of number of workers
	Replicas int32 `json:"replicas,omitempty"`
	// Resources is the resource requests and limits for the pods.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

func (s *WorkerSpec) validate(fp *field.Path) field.ErrorList {
	return field.ErrorList{}
}

//GCSSpec defines the atributed needed to sync from a git repo
type GCSSpec struct {
	// Bucket describes the GCS bucket
	Bucket string `json:"bucket,omitempty"`
	// Once syncs initially and quits (use init container instead of sidecar)
	Once bool `json:"once,omitempty"`
}

func (s *GCSSpec) validate(fp *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if s == nil {
		return errs
	}
	if s.Bucket == "" {
		errs = append(errs, field.Required(fp.Child("bucket"), "bucket required"))
	}
	if s.Once == true {
		errs = append(errs, field.NotSupported(fp.Child("once"), "true", []string{}))
	}
	return errs
}

//GitSpec defines the atributed needed to sync from a git repo
type GitSpec struct {
	// Repo describes the http/ssh uri for git repo
	Repo string `json:"repo,"`
	// Branch describes the branch name to be synced
	Branch string `json:"branch,omitempty"`
	// Rev is the git hash to be used for syncing
	Rev string `json:"rev,omitempty"`
	// User for git access
	User string `json:"user,omitempty"`
	// Once syncs initially and quits (use init container instead of sidecar)
	Once bool `json:"once,omitempty"`
	// Reference to git credentials (user, password, ssh etc)
	CredSecretRef *corev1.LocalObjectReference `json:"cred,omitempty"`
}

func (s *GitSpec) validate(fp *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if s == nil {
		return errs
	}

	if s.Repo == "" {
		errs = append(errs, field.Required(fp.Child("repo"), "repo required"))
	}
	if s.CredSecretRef != nil && s.CredSecretRef.Name == "" {
		errs = append(errs, field.Required(fp.Child("cred", "name"), "name missing"))
	}
	//errs = append(errs, field.NotSupported(fp.Child("cred"), "", []string{}))
	return errs
}

// DagSpec defines where the DAGs are located and how to access them
type DagSpec struct {
	// DagSubdir is the directory under source where the dags are present
	DagSubdir string `json:"subdir,omitempty"`
	// GitSpec defines details to pull DAGs from a git repo using
	// github.com/kubernetes/git-sync sidecar
	Git *GitSpec `json:"git,omitempty"`
	// NfsPVSpec
	NfsPV *corev1.PersistentVolumeClaim `json:"nfspv,omitempty"`
	// Storage has s3 compatible storage spec for copying files from
	Storage *StorageSpec `json:"storage,omitempty"`
	// Gcs config which uses storage spec
	GCS *GCSSpec `json:"gcs,omitempty"`
}

func (s *DagSpec) validate(fp *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if s == nil {
		return errs
	}
	if s.NfsPV != nil {
		errs = append(errs, field.NotSupported(fp.Child("nfspv"), "", []string{}))
	}
	if s.Storage != nil {
		errs = append(errs, field.NotSupported(fp.Child("storage"), "", []string{}))
	}
	errs = append(errs, s.Git.validate(fp.Child("git"))...)
	errs = append(errs, s.GCS.validate(fp.Child("git"))...)
	return errs
}

// ClusterConfig is used to capture the config for Airflow
type ClusterConfig struct {
	// Airflow defines a list of kv pairs that describe env variables injected into the nodes
	// +optional
	AirflowEnv map[string]string `json:"airflow,omitempty"`
}

// AirflowClusterSpec defines the desired state of AirflowCluster
type AirflowClusterSpec struct {
	// Selector for fitting pods to nodes whose labels match the selector.
	// https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Define scheduling constraints for pods.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Custom annotations to be added to the pods.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Custom labels to be added to the pods.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Airflow Executor desired: local,celery,kubernetes
	// +optional
	Executor string `json:"executor,omitempty"`
	// Airflow config as env list
	// +optional
	Config ClusterConfig `json:"config,omitempty"`
	// Spec for Redis component.
	// +optional
	Redis *RedisSpec `json:"redis,omitempty"`
	// Spec for Airflow Scheduler component.
	// +optional
	Scheduler *SchedulerSpec `json:"scheduler,omitempty"`
	// Spec for Airflow Workers
	// +optional
	Worker *WorkerSpec `json:"worker,omitempty"`
	// Spec for Airflow UI component.
	// +optional
	UI *AirflowUISpec `json:"ui,omitempty"`
	// Spec for Flower component.
	// +optional
	Flower *FlowerSpec `json:"flower,omitempty"`
	// Spec for DAG source and location
	// +optional
	DAGs *DagSpec `json:"dags,omitempty"`
	// AirflowBaseRef is a reference to the AirflowBase CR
	AirflowBaseRef *corev1.LocalObjectReference `json:"airflowbase,omitempty"`
}

// SchedulerStatus defines the observed state of Airflow Scheduler
type SchedulerStatus struct {
	// DagCount is a count of number of Dags observed
	DagCount int32 `json:"dagcount,omitempty"`
	// RunCount is a count of number of Dag Runs observed
	RunCount int32 `json:"runcount,omitempty"`
}

// AirflowClusterStatus defines the observed state of AirflowCluster
type AirflowClusterStatus struct {
	status.Meta `json:",inline"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AirflowCluster represents the Airflow Schduler and workers for a single DAG folder
// function. At a minimum they need a SQL service (MySQL or SQLProxy) and Airflow UI.
// In addition for an installation with minimal external dependencies, NFS and Airflow UI
// are also added.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=airflowclusters
type AirflowCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AirflowClusterSpec   `json:"spec,omitempty"`
	Status AirflowClusterStatus `json:"status,omitempty"`
}

// Helper functions for the resources

// ApplyDefaults the AirflowCluster
func (b *AirflowCluster) ApplyDefaults() {
	if b.Spec.Redis != nil {
		if b.Spec.Redis.Image == "" {
			b.Spec.Redis.Image = defaultRedisImage
		}
		if b.Spec.Redis.Version == "" {
			b.Spec.Redis.Version = defaultRedisVersion
		}
	}
	if b.Spec.Scheduler != nil {
		if b.Spec.Scheduler.Image == "" {
			b.Spec.Scheduler.Image = defaultSchedulerImage
		}
		if b.Spec.Scheduler.Version == "" {
			b.Spec.Scheduler.Version = defaultSchedulerVersion
		}
		if b.Spec.Scheduler.DBName == "" {
			b.Spec.Scheduler.DBName = string(RandomAlphanumericString(16))
		}
		if b.Spec.Scheduler.DBUser == "" {
			b.Spec.Scheduler.DBUser = string(RandomAlphanumericString(16))
		}
	}
	if b.Spec.UI != nil {
		if b.Spec.UI.Image == "" {
			b.Spec.UI.Image = defaultUIImage
		}
		if b.Spec.UI.Version == "" {
			b.Spec.UI.Version = defaultUIVersion
		}
		if b.Spec.UI.Replicas == 0 {
			b.Spec.UI.Replicas = 1
		}
	}
	if b.Spec.Flower != nil {
		if b.Spec.Flower.Image == "" {
			b.Spec.Flower.Image = defaultFlowerImage
		}
		if b.Spec.Flower.Version == "" {
			b.Spec.Flower.Version = defaultFlowerVersion
		}
		if b.Spec.Flower.Replicas == 0 {
			b.Spec.Flower.Replicas = 1
		}
	}
	if b.Spec.Executor == "" {
		b.Spec.Executor = defaultExecutor
	}
	if b.Spec.Worker != nil {
		if b.Spec.Worker.Image == "" {
			b.Spec.Worker.Image = defaultWorkerImage
		}
		if b.Spec.Worker.Version == "" {
			b.Spec.Worker.Version = defaultWorkerVersion
		}
		if b.Spec.Worker.Replicas == 0 {
			b.Spec.Worker.Replicas = 1
		}
		if b.Spec.Executor == ExecutorK8s {
			b.Spec.Worker.Replicas = 0
		}
	}
	if b.Spec.DAGs != nil {
		if b.Spec.DAGs.Git != nil {
			if b.Spec.DAGs.Git.Branch == "" {
				b.Spec.DAGs.Git.Branch = defaultBranch
			}
		}
	}
}

// UpdateRsrcStatus records status or error in status
func (b *AirflowCluster) UpdateRsrcStatus(status interface{}, err error) bool {
	esstatus := status.(*AirflowClusterStatus)
	if status != nil {
		b.Status = *esstatus
	}

	if err != nil {
		b.Status.SetError("ErrorSeen", err.Error())
	} else {
		b.Status.ClearError()
	}
	// TODO use err
	return true
}

// Validate the AirflowCluster
func (b *AirflowCluster) Validate() error {
	errs := field.ErrorList{}
	spec := field.NewPath("spec")

	errs = append(errs, b.Spec.Redis.validate(spec.Child("redis"))...)
	errs = append(errs, b.Spec.Scheduler.validate(spec.Child("scehduler"))...)
	errs = append(errs, b.Spec.Worker.validate(spec.Child("worker"))...)
	errs = append(errs, b.Spec.DAGs.validate(spec.Child("dags"))...)
	errs = append(errs, b.Spec.UI.validate(spec.Child("ui"))...)
	errs = append(errs, b.Spec.Flower.validate(spec.Child("flower"))...)

	allowed := false
	for _, executor := range allowedExecutors {
		if executor == b.Spec.Executor {
			allowed = true
		}
	}
	if !allowed {
		errs = append(errs, field.NotSupported(spec.Child("executor"), b.Spec.Executor, allowedExecutors))
	}

	if b.Spec.Scheduler == nil {
		errs = append(errs, field.Required(spec.Child("scheduler"), "scheduler required"))
	}

	if b.Spec.Executor == ExecutorCelery {
		if b.Spec.Redis == nil {
			errs = append(errs, field.Required(spec.Child("redis"), "redis required for Celery executor"))
		}
		if b.Spec.Worker == nil {
			errs = append(errs, field.Required(spec.Child("worker"), "worker required for Celery executor"))
		}
	}
	if b.Spec.Executor == ExecutorK8s {
		if b.Spec.Worker == nil {
			errs = append(errs, field.Required(spec.Child("worker"), "worker required for Celery executor"))
		}
	}

	if b.Spec.Flower != nil {
		if b.Spec.Executor != ExecutorCelery {
			errs = append(errs, field.Required(spec.Child("executor"), "celery executor required for Flower"))
		}
	}

	if b.Spec.AirflowBaseRef == nil {
		errs = append(errs, field.Required(spec.Child("airflowbase"), "airflowbase reference missing"))
	} else if b.Spec.AirflowBaseRef.Name == "" {
		errs = append(errs, field.Required(spec.Child("airflowbase", "name"), "name missing"))
	}

	return errs.ToAggregate()
}

// Components returns components for this resource
func (b *AirflowCluster) Components() []component.Component {
	c := []component.Component{}
	if b.Spec.Redis != nil {
		c = append(c, component.Component{
			Handle:   b.Spec.Redis,
			Name:     ValueAirflowComponentRedis,
			CR:       b,
			OwnerRef: b.OwnerRef(),
		})
	}
	if b.Spec.Flower != nil {
		c = append(c, component.Component{
			Handle:   b.Spec.Flower,
			Name:     ValueAirflowComponentFlower,
			CR:       b,
			OwnerRef: b.OwnerRef(),
		})
	}
	if b.Spec.Scheduler != nil {
		c = append(c, component.Component{
			Handle:   b.Spec.Scheduler,
			Name:     ValueAirflowComponentScheduler,
			CR:       b,
			OwnerRef: b.OwnerRef(),
		})
	}
	if b.Spec.UI != nil {
		c = append(c, component.Component{
			Handle:   b.Spec.UI,
			Name:     ValueAirflowComponentUI,
			CR:       b,
			OwnerRef: b.OwnerRef(),
		})
	}
	if b.Spec.Worker != nil {
		c = append(c, component.Component{
			Handle:   b.Spec.Worker,
			Name:     ValueAirflowComponentWorker,
			CR:       b,
			OwnerRef: b.OwnerRef(),
		})
	}
	c = append(c, component.Component{
		Handle:   b,
		Name:     ValueAirflowComponentCluster,
		CR:       b,
		OwnerRef: b.OwnerRef(),
	})
	return c
}

// OwnerRef returns owner ref object with the component's resource as owner
func (b *AirflowCluster) OwnerRef() []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(b, schema.GroupVersionKind{
			Group:   SchemeGroupVersion.Group,
			Version: SchemeGroupVersion.Version,
			Kind:    "AirflowCluster",
		}),
	}
}

// NewRsrc - return a new resource object
func (b *AirflowCluster) NewRsrc() cr.Handle {
	return &AirflowCluster{}
}

// NewStatus - return a  resource status object
func (b *AirflowCluster) NewStatus() interface{} {
	s := b.Status.DeepCopy()
	s.ComponentList = status.ComponentList{}
	return s
}

// StatusDiffers returns True if there is a change in status
func (b *AirflowCluster) StatusDiffers(new AirflowClusterStatus) bool {
	return true
}

// NewAirflowCluster return a defaults filled AirflowCluster object
func NewAirflowCluster(name, namespace, executor, base string, dags *DagSpec) *AirflowCluster {
	c := AirflowCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{},
			Namespace: namespace,
		},
	}
	c.Spec = AirflowClusterSpec{}
	c.Spec.Executor = executor
	c.Spec.Scheduler = &SchedulerSpec{}
	c.Spec.UI = &AirflowUISpec{}
	if executor == ExecutorCelery {
		c.Spec.Redis = &RedisSpec{}
		c.Spec.Worker = &WorkerSpec{}
		c.Spec.Flower = &FlowerSpec{}
	}
	c.Spec.DAGs = dags
	c.Spec.AirflowBaseRef = &corev1.LocalObjectReference{Name: base}
	c.ApplyDefaults()
	return &c
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AirflowClusterList contains a list of AirflowCluster
type AirflowClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AirflowCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AirflowCluster{}, &AirflowClusterList{})
}
