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
	"encoding/base64"
	"fmt"
	app "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"math/rand"
	"sigs.k8s.io/kubesdk/pkg/application"
	"sigs.k8s.io/kubesdk/pkg/component"
	"sigs.k8s.io/kubesdk/pkg/resource"
	"sigs.k8s.io/kubesdk/pkg/resource/manager/k8s"
	"sort"
	"strconv"
	"time"
)

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=airflow.k8s.io,resources=airflowbases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=airflow.k8s.io,resources=airflowclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.k8s.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch;create;update;patch;delete
// constants defining field values
const (
	ControllerVersion = "0.1"

	PasswordCharNumSpace = "abcdefghijklmnopqrstuvwxyz0123456789"
	PasswordCharSpace    = "abcdefghijklmnopqrstuvwxyz"

	ActionCheck  = "check"
	ActionCreate = "create"
	ActionDelete = "delete"

	LabelAirflowCR                 = "airflow-cr"
	ValueAirflowCRBase             = "airflow-base"
	ValueAirflowCRCluster          = "airflow-cluster"
	LabelAirflowCRName             = "airflow-cr-name"
	LabelAirflowComponent          = "airflow-component"
	ValueAirflowComponentMySQL     = "mysql"
	ValueAirflowComponentPostgres  = "postgres"
	ValueAirflowComponentSQLProxy  = "sqlproxy"
	ValueAirflowComponentBase      = "base"
	ValueAirflowComponentCluster   = "cluster"
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

	TemplatePath = "templates/"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
)

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
		result[i] = PasswordCharNumSpace[random.Intn(len(PasswordCharNumSpace))]
	}
	result[0] = PasswordCharSpace[random.Intn(len(PasswordCharSpace))]
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

func rsrcName(name string, component string, suffix string) string {
	return name + "-" + component + suffix
}

func (r *AirflowCluster) dependantResources() *resource.Bag {
	rsrc := &resource.Bag{}
	rsrc.Add(k8s.ReferredItem(&AirflowBase{}, r.Spec.AirflowBaseRef.Name, r.Namespace))
	return rsrc
}

func (r *AirflowCluster) getAirflowPrometheusEnv(base *AirflowBase) []corev1.EnvVar {
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	ap := "AIRFLOW_PROMETHEUS_"
	apd := ap + "DATABASE_"
	backend := "mysql"
	port := "3306"
	if base.Spec.Postgres != nil {
		backend = "postgres"
		port = "5432"
	}
	env := []corev1.EnvVar{
		{Name: ap + "LISTEN_ADDR", Value: ":9112"},
		{Name: apd + "BACKEND", Value: backend},
		{Name: apd + "HOST", Value: sqlSvcName},
		{Name: apd + "PORT", Value: port},
		{Name: apd + "USER", Value: r.Spec.Scheduler.DBUser},
		{Name: apd + "PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: apd + "NAME", Value: r.Spec.Scheduler.DBName},
	}
	return env
}

func (r *AirflowCluster) getAirflowEnv(saName string, base *AirflowBase) []corev1.EnvVar {
	sp := r.Spec
	sqlSvcName := rsrcName(sp.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	redisSecret := rsrcName(r.Name, ValueAirflowComponentRedis, "")
	schedulerConfigmap := rsrcName(r.Name, ValueAirflowComponentScheduler, "")
	redisSvcName := redisSecret
	dagFolder := AirflowDagsBase
	if sp.DAGs != nil {
		if sp.DAGs.Git != nil {
			dagFolder = AirflowDagsBase + "/" + GitSyncDestDir + "/" + sp.DAGs.DagSubdir
		} else if sp.DAGs.GCS != nil {
			dagFolder = AirflowDagsBase + "/" + GCSSyncDestDir + "/" + sp.DAGs.DagSubdir
		}
	}
	dbType := "mysql"
	if base.Spec.Postgres != nil {
		dbType = "postgres"
	}
	env := []corev1.EnvVar{
		{Name: "EXECUTOR", Value: sp.Executor},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: afc + "DAGS_FOLDER", Value: dagFolder},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "SQL_USER", Value: sp.Scheduler.DBUser},
		{Name: "SQL_DB", Value: sp.Scheduler.DBName},
		{Name: "DB_TYPE", Value: dbType},
	}
	if sp.Executor == ExecutorK8s {
		env = append(env, []corev1.EnvVar{
			{Name: afk + "WORKER_CONTAINER_REPOSITORY", Value: sp.Worker.Image},
			{Name: afk + "WORKER_CONTAINER_TAG", Value: sp.Worker.Version},
			{Name: afk + "WORKER_CONTAINER_IMAGE_PULL_POLICY", Value: "IfNotPresent"},
			{Name: afk + "DELETE_WORKER_PODS", Value: "True"},
			{Name: afk + "NAMESPACE", Value: r.Namespace},
			{Name: afk + "AIRFLOW_CONFIGMAP", Value: schedulerConfigmap},
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

	// Do sorted key scan. To store the keys in slice in sorted order
	var keys []string
	for k := range sp.Config.AirflowEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		env = append(env, corev1.EnvVar{Name: k, Value: sp.Config.AirflowEnv[k]})
	}

	return env
}

func (r *AirflowCluster) addAirflowContainers(ss *appsv1.StatefulSet) {
	if r.Spec.DAGs != nil {
		init, dagContainer := r.Spec.DAGs.container("dags-data")
		if init {
			ss.Spec.Template.Spec.InitContainers = append(ss.Spec.Template.Spec.InitContainers, dagContainer)
		} else {
			ss.Spec.Template.Spec.Containers = append(ss.Spec.Template.Spec.Containers, dagContainer)
		}
	}
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
SHOW GRANTS FOR $(SQL_USER);
EOSQL
`},
		},
	}
	ss.Spec.Template.Spec.InitContainers = append(containers, ss.Spec.Template.Spec.InitContainers...)
}

func (r *AirflowCluster) addPostgresUserDBContainer(ss *appsv1.StatefulSet) {
	sqlRootSecret := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
	sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
	env := []corev1.EnvVar{
		{Name: "SQL_ROOT_PASSWORD", ValueFrom: envFromSecret(sqlRootSecret, "rootpassword")},
		{Name: "SQL_DB", Value: r.Spec.Scheduler.DBName},
		{Name: "SQL_USER", Value: r.Spec.Scheduler.DBUser},
		{Name: "SQL_PASSWORD", ValueFrom: envFromSecret(sqlSecret, "password")},
		{Name: "SQL_HOST", Value: sqlSvcName},
		{Name: "DB_TYPE", Value: "postgres"},
	}
	containers := []corev1.Container{
		{
			Name:    "postgres-dbcreate",
			Image:   defaultPostgresImage + ":" + defaultPostgresVersion,
			Env:     env,
			Command: []string{"/bin/bash"},
			Args: []string{"-c", `
PGPASSWORD=$(SQL_ROOT_PASSWORD) psql -h $SQL_HOST -U airflow -d testdb -c "CREATE DATABASE $(SQL_DB)";
PGPASSWORD=$(SQL_ROOT_PASSWORD) psql -h $SQL_HOST -U airflow -d testdb -c "CREATE USER $(SQL_USER) WITH ENCRYPTED PASSWORD '$(SQL_PASSWORD)'; GRANT ALL PRIVILEGES ON DATABASE $(SQL_DB) TO $(SQL_USER)"
`},
		},
	}
	ss.Spec.Template.Spec.InitContainers = append(containers, ss.Spec.Template.Spec.InitContainers...)
}

// ------------------------------ MYSQL  ---------------------------------------

type commonTmplValue struct {
	Name        string
	Namespace   string
	SecretName  string
	SvcName     string
	Base        *AirflowBase
	Cluster     *AirflowCluster
	Labels      component.KVMap
	Selector    component.KVMap
	Ports       map[string]string
	Secret      map[string]string
	PDBMinAvail string
	Expected    *resource.Bag
	SQLConn     string
}

func tmplSecret(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"secret.yaml", v, &corev1.SecretList{})
}

func tmplServiceaccount(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"serviceaccount.yaml", v, &corev1.ServiceAccountList{})
}

func tmplRolebinding(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"rolebinding.yaml", v, &rbacv1.RoleBindingList{})
}

func tmplsvc(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"svc.yaml", v, &corev1.ServiceList{})
}

func tmpWorkerSvc(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"worker-svc.yaml", v, &corev1.ServiceList{})
}

func tmplpodDisruption(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"pdb.yaml", v, &policyv1.PodDisruptionBudgetList{})
}

func (s *MySQLSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"mysql-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Base.Spec.MySQL.Resources
		if r.Base.Spec.MySQL.VolumeClaimTemplate != nil {
			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*r.Base.Spec.MySQL.VolumeClaimTemplate}
		}
	}
	return o, err
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *MySQLSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowBase)
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentMySQL, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		SvcName:    rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		Base:       r,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		Ports:      map[string]string{"mysql": "3306"},
		Secret: map[string]string{
			"password":     base64.StdEncoding.EncodeToString(RandomAlphanumericString(16)),
			"rootpassword": base64.StdEncoding.EncodeToString(RandomAlphanumericString(16)),
		},
		PDBMinAvail: "100%",
	}

	for _, fn := range []resource.GetItemFn{s.sts, tmplsvc, tmplpodDisruption, tmplSecret} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// differs returns true if the resource needs to be updated
func differs(expected resource.Item, observed resource.Item) bool {
	switch expected.Obj.(*k8s.Object).Obj.(type) {
	case *corev1.ServiceAccount:
		// Dont update a SA
		return false
	case *corev1.Secret:
		// Dont update a secret
		return false
	}
	return true
}

// Differs returns true if the resource needs to be updated
func (s *MySQLSpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *MySQLSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

// ------------------------------ POSTGRES  ---------------------------------------

func (s *PostgresSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"postgres-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Base.Spec.Postgres.Resources
		if r.Base.Spec.Postgres.VolumeClaimTemplate != nil {
			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*r.Base.Spec.Postgres.VolumeClaimTemplate}
		}
	}
	return o, err
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *PostgresSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowBase)
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentPostgres, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		SvcName:    rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		Base:       r,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		Ports:      map[string]string{"postgres": "5432"},
		Secret: map[string]string{
			"password":     base64.StdEncoding.EncodeToString(RandomAlphanumericString(16)),
			"rootpassword": base64.StdEncoding.EncodeToString(RandomAlphanumericString(16)),
		},
		PDBMinAvail: "100%",
	}

	for _, fn := range []resource.GetItemFn{tmplsvc, tmplpodDisruption, tmplSecret, s.sts} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// Differs returns true if the resource needs to be updated
func (s *PostgresSpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *PostgresSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

// ------------------------------ Airflow UI ---------------------------------------

// DependentResources - return dependant resources
func (s *AirflowUISpec) DependentResources(rsrc interface{}) *resource.Bag {
	r := rsrc.(*AirflowCluster)
	return r.dependantResources()
}

// ExpectedResources returns the list of resource/name for those resources created by
func (s *AirflowUISpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowCluster)
	b := k8s.GetItem(dependent, &AirflowBase{}, r.Spec.AirflowBaseRef.Name, r.Namespace)
	base := b.(*AirflowBase)
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentUI, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentUI, ""),
		Cluster:    r,
		Base:       base,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		Ports:      map[string]string{"web": "8080"},
		Secret: map[string]string{
			"password": base64.StdEncoding.EncodeToString(RandomAlphanumericString(16)),
		},
	}

	for _, fn := range []resource.GetItemFn{s.sts, tmplSecret} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}

	return resources, nil
}

// Differs returns true if the resource needs to be updated
func (s *AirflowUISpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *AirflowUISpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

func (s *AirflowUISpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"ui-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Cluster.Spec.UI.Resources
		sts.Spec.Template.Spec.Containers[0].Env = r.Cluster.getAirflowEnv(sts.Name, r.Base)
		r.Cluster.addAirflowContainers(sts)
		if r.Base.Spec.Postgres != nil {
			r.Cluster.addPostgresUserDBContainer(sts)
		} else {
			r.Cluster.addMySQLUserDBContainer(sts)
		}
	}
	return o, err
}

// ------------------------------ NFSStoreSpec ---------------------------------------

// ExpectedResources returns the list of resource/name for those resources created by
func (s *NFSStoreSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowBase)
	var ngdata = commonTmplValue{
		Name:        rsrcName(r.Name, ValueAirflowComponentNFS, ""),
		Namespace:   r.Namespace,
		SvcName:     rsrcName(r.Name, ValueAirflowComponentNFS, ""),
		Base:        r,
		Labels:      rsrclabels,
		Selector:    rsrclabels,
		Ports:       map[string]string{"nfs": "2049", "mountd": "20048", "rpcbind": "111"},
		PDBMinAvail: "100%",
	}

	for _, fn := range []resource.GetItemFn{tmplsvc, tmplpodDisruption, s.sts} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

func (s *NFSStoreSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"nfs-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Base.Spec.Storage.Resources
		if r.Base.Spec.Storage.Volume != nil {
			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*r.Base.Spec.Storage.Volume}
		}
	}
	return o, err
}

// Differs returns true if the resource needs to be updated
func (s *NFSStoreSpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *NFSStoreSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

// ------------------------------ SQLProxy ---------------------------------------

func (s *SQLProxySpec) sts(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"sqlproxy-sts.yaml", v, &appsv1.StatefulSetList{})
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *SQLProxySpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowBase)
	name := rsrcName(r.Name, ValueAirflowComponentSQL, "")
	resources.Add(k8s.ReferredItem(&corev1.Secret{}, name, r.Namespace))

	var ngdata = commonTmplValue{
		Name:      rsrcName(r.Name, ValueAirflowComponentSQLProxy, ""),
		Namespace: r.Namespace,
		SvcName:   rsrcName(r.Name, ValueAirflowComponentSQL, ""),
		Base:      r,
		Labels:    rsrclabels,
		Selector:  rsrclabels,
		Ports:     map[string]string{"sqlproxy": "3306"},
	}

	for _, fn := range []resource.GetItemFn{tmplsvc, s.sts} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// Differs returns true if the resource needs to be updated
func (s *SQLProxySpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *SQLProxySpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowBaseStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

// ------------------------------ RedisSpec ---------------------------------------

func (s *RedisSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"redis-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Cluster.Spec.Redis.Resources
		if r.Cluster.Spec.Redis.VolumeClaimTemplate != nil {
			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*r.Cluster.Spec.Redis.VolumeClaimTemplate}
		}
	}
	return o, err
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *RedisSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowCluster)
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentRedis, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentRedis, ""),
		SvcName:    rsrcName(r.Name, ValueAirflowComponentRedis, ""),
		Cluster:    r,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		Ports:      map[string]string{"redis": "6379"},
		Secret: map[string]string{
			"password": base64.StdEncoding.EncodeToString(RandomAlphanumericString(16)),
		},
		PDBMinAvail: "100%",
	}

	for _, fn := range []resource.GetItemFn{tmplsvc, tmplpodDisruption, tmplSecret, s.sts} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// Differs returns true if the resource needs to be updated
func (s *RedisSpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *RedisSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
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
			{Name: "GIT_SYNC_PASSWORD",
				ValueFrom: envFromSecret(s.CredSecretRef.Name, "password")},
			{Name: "GIT_SYNC_USERNAME", Value: s.User},
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
				MountPath: "/tmp/git",
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

func (s *SchedulerSpec) configmap(v interface{}) (*resource.Item, error) {
	return k8s.ItemFromFile(TemplatePath+"airflow-configmap.yaml", v, &corev1.ConfigMapList{})
}

func (s *SchedulerSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"scheduler-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		if r.Cluster.Spec.Executor == ExecutorK8s {
			sts.Spec.Template.Spec.ServiceAccountName = sts.Name
		}
		sts.Spec.Template.Spec.Containers[0].Resources = r.Cluster.Spec.Scheduler.Resources
		sts.Spec.Template.Spec.Containers[0].Env = r.Cluster.getAirflowEnv(sts.Name, r.Base)
		sts.Spec.Template.Spec.Containers[1].Env = r.Cluster.getAirflowPrometheusEnv(r.Base)
		r.Cluster.addAirflowContainers(sts)
	}
	return o, err
}

// DependentResources - return dependant resources
func (s *SchedulerSpec) DependentResources(rsrc interface{}) *resource.Bag {
	r := rsrc.(*AirflowCluster)
	resources := r.dependantResources()
	if r.Spec.Executor == ExecutorK8s {
		sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
		resources.Add(k8s.ReferredItem(&corev1.Secret{}, sqlSecret, r.Namespace))
	}
	return resources
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *SchedulerSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowCluster)
	b := k8s.GetItem(dependent, &AirflowBase{}, r.Spec.AirflowBaseRef.Name, r.Namespace)
	base := b.(*AirflowBase)
	if r.Spec.DAGs != nil {
		git := r.Spec.DAGs.Git
		if git != nil && git.CredSecretRef != nil {
			resources.Add(k8s.ReferredItem(&corev1.Secret{}, git.CredSecretRef.Name, r.Namespace))
		}
	}

	if r.Spec.Executor == ExecutorK8s {
		sqlSvcName := rsrcName(r.Spec.AirflowBaseRef.Name, ValueAirflowComponentSQL, "")
		sqlSecret := rsrcName(r.Name, ValueAirflowComponentUI, "")
		se := k8s.GetItem(dependent, &corev1.Secret{}, sqlSecret, r.Namespace)
		secret := se.(*corev1.Secret)

		dbPrefix := "mysql"
		port := "3306"
		if base.Spec.Postgres != nil {
			dbPrefix = "postgresql+psycopg2"
			port = "5432"
		}
		conn := dbPrefix + "://" + s.DBUser + ":" + string(secret.Data["password"]) + "@" + sqlSvcName + ":" + port + "/" + s.DBName

		var ngdata = commonTmplValue{
			Name:      rsrcName(r.Name, ValueAirflowComponentScheduler, ""),
			Namespace: r.Namespace,
			Cluster:   r,
			Labels:    rsrclabels,
			SQLConn:   conn,
		}
		for _, fn := range []resource.GetItemFn{s.configmap} {
			rinfo, err := fn(&ngdata)
			if err != nil {
				return nil, err
			}
			resources.Add(*rinfo)
		}
	}
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentScheduler, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentScheduler, ""),
		Cluster:    r,
		Base:       base,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		SQLConn:    "",
	}

	for _, fn := range []resource.GetItemFn{s.sts, tmplServiceaccount, tmplRolebinding} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// Differs returns true if the resource needs to be updated
func (s *SchedulerSpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *SchedulerSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

// ------------------------------ Worker -

func (s *WorkerSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"worker-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Cluster.Spec.Worker.Resources
		sts.Spec.Template.Spec.Containers[0].Env = r.Cluster.getAirflowEnv(sts.Name, r.Base)
		r.Cluster.addAirflowContainers(sts)
	}
	return o, err
}

// DependentResources - return dependant resources
func (s *WorkerSpec) DependentResources(rsrc interface{}) *resource.Bag {
	r := rsrc.(*AirflowCluster)
	return r.dependantResources()
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (s *WorkerSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowCluster)
	b := k8s.GetItem(dependent, &AirflowBase{}, r.Spec.AirflowBaseRef.Name, r.Namespace)
	base := b.(*AirflowBase)
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentWorker, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentWorker, ""),
		SvcName:    rsrcName(r.Name, ValueAirflowComponentWorker, ""),
		Cluster:    r,
		Base:       base,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		Ports:      map[string]string{"wlog": "8793"},
	}

	for _, fn := range []resource.GetItemFn{tmpWorkerSvc, s.sts} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *WorkerSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

// ------------------------------ Flower ---------------------------------------

// DependentResources - return dependant resources
func (s *FlowerSpec) DependentResources(rsrc interface{}) *resource.Bag {
	r := rsrc.(*AirflowCluster)
	return r.dependantResources()
}

// ExpectedResources returns the list of resource/name for those resources created by
func (s *FlowerSpec) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)
	r := rsrc.(*AirflowCluster)
	b := k8s.GetItem(dependent, &AirflowBase{}, r.Spec.AirflowBaseRef.Name, r.Namespace)
	base := b.(*AirflowBase)
	var ngdata = commonTmplValue{
		Name:       rsrcName(r.Name, ValueAirflowComponentFlower, ""),
		Namespace:  r.Namespace,
		SecretName: rsrcName(r.Name, ValueAirflowComponentFlower, ""),
		Cluster:    r,
		Base:       base,
		Labels:     rsrclabels,
		Selector:   rsrclabels,
		Ports:      map[string]string{"flower": "5555"},
	}

	for _, fn := range []resource.GetItemFn{s.sts} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// Differs returns true if the resource needs to be updated
func (s *FlowerSpec) Differs(expected resource.Item, observed resource.Item) bool {
	return differs(expected, observed)
}

// UpdateComponentStatus use reconciled objects to update component status
func (s *FlowerSpec) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s != nil {
		stts := statusi.(*AirflowClusterStatus)
		ready := stts.ComponentMeta.UpdateStatus(reconciled.ByType(k8s.Type))
		stts.Meta.UpdateStatus(&ready, err)
	}
	return period
}

func (s *FlowerSpec) sts(v interface{}) (*resource.Item, error) {
	r := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"flower-sts.yaml", v, &appsv1.StatefulSetList{})
	if err == nil {
		sts := o.Obj.(*k8s.Object).Obj.(*appsv1.StatefulSet)
		sts.Spec.Template.Spec.Containers[0].Resources = r.Cluster.Spec.Flower.Resources
		sts.Spec.Template.Spec.Containers[0].Env = r.Cluster.getAirflowEnv(sts.Name, r.Base)
		r.Cluster.addAirflowContainers(sts)
	}
	return o, err
}

// ---------------- Global AirflowCluster component -------------------------

// Mutate - mutate expected
func (r *AirflowCluster) Mutate(rsrc interface{}, rsrclabels map[string]string, status interface{}, expected, dependent, observed *resource.Bag) (*resource.Bag, error) {
	return expected, nil
}

func (r *AirflowCluster) appcrd(v interface{}) (*resource.Item, error) {
	value := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"cluster-application.yaml", v, nil)

	if err == nil {
		ao := application.Application{Application: *o.Obj.(*k8s.Object).Obj.(*app.Application)}
		o = ao.SetComponentGK(value.Expected).Item()
	}
	return o, err
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (r *AirflowCluster) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)

	selectors := make(map[string]string)
	for k, v := range rsrclabels {
		selectors[k] = v
	}
	delete(selectors, component.LabelComponent)
	var ngdata = commonTmplValue{
		Name:      rsrcName(r.Name, ValueAirflowComponentCluster, ""),
		Namespace: r.Namespace,
		Labels:    rsrclabels,
		Selector:  selectors,
		Expected:  aggregated,
	}

	for _, fn := range []resource.GetItemFn{r.appcrd} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// UpdateComponentStatus use reconciled objects to update component status
func (r *AirflowCluster) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	return period
}

// ---------------- Global AirflowBase component -------------------------

// Mutate - mutate expected
func (r *AirflowBase) Mutate(rsrc interface{}, rsrclabels map[string]string, status interface{}, expected, dependent, observed *resource.Bag) (*resource.Bag, error) {
	return expected, nil
}

func (r *AirflowBase) appcrd(v interface{}) (*resource.Item, error) {
	value := v.(*commonTmplValue)
	o, err := k8s.ItemFromFile(TemplatePath+"base-application.yaml", v, nil)

	if err == nil {
		ao := application.Application{Application: *o.Obj.(*k8s.Object).Obj.(*app.Application)}
		o = ao.SetComponentGK(value.Expected).Item()
	}
	return o, err
}

// ExpectedResources returns the list of resource/name for those resources created by
// the operator for this spec and those resources referenced by this operator.
// Mark resources as owned, referred
func (r *AirflowBase) ExpectedResources(rsrc interface{}, rsrclabels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error) {
	var resources *resource.Bag = new(resource.Bag)

	selectors := make(map[string]string)
	for k, v := range rsrclabels {
		selectors[k] = v
	}
	delete(selectors, component.LabelComponent)
	var ngdata = commonTmplValue{
		Name:      rsrcName(r.Name, ValueAirflowComponentBase, ""),
		Namespace: r.Namespace,
		Labels:    rsrclabels,
		Selector:  selectors,
		Expected:  aggregated,
	}

	for _, fn := range []resource.GetItemFn{r.appcrd} {
		rinfo, err := fn(&ngdata)
		if err != nil {
			return nil, err
		}
		resources.Add(*rinfo)
	}
	return resources, nil
}

// UpdateComponentStatus use reconciled objects to update component status
func (r *AirflowBase) UpdateComponentStatus(rsrci, statusi interface{}, reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	return period
}
