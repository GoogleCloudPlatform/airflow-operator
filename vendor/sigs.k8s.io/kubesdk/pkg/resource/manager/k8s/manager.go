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

package k8s

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubesdk/pkg/resource"
	"sigs.k8s.io/kubesdk/pkg/resource/manager"
	"strings"
	"text/template"
)

// constants
const (
	Type = "k8s"
)

// RsrcManager - complies with resource manager interface
type RsrcManager struct {
	name   string
	client client.Client
	scheme *runtime.Scheme
}

// NewRsrcManager returns nil manager
func NewRsrcManager() *RsrcManager {
	return &RsrcManager{}
}

// WithName adds name
func (rm *RsrcManager) WithName(v string) *RsrcManager {
	rm.name = v
	return rm
}

// WithClient attaches client
func (rm *RsrcManager) WithClient(v client.Client) *RsrcManager {
	rm.client = v
	return rm
}

// WithScheme adds scheme
func (rm *RsrcManager) WithScheme(v *runtime.Scheme) *RsrcManager {
	rm.scheme = v
	return rm
}

// GetScheme gets scheme
func (rm *RsrcManager) GetScheme() *runtime.Scheme {
	return rm.scheme
}

// Object - K8s object
type Object struct {
	// Obj refers to the resource object  can be: sts, service, secret, pvc, ..
	Obj metav1.Object
	// ObjList refers to the list of resource objects
	ObjList metav1.ListInterface
}

func isReferringSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}
	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}
	return aGV == bGV && a.Kind == b.Kind && a.Name == b.Name
}

// SetOwnerReferences - return name string
func (o *Object) SetOwnerReferences(ref *metav1.OwnerReference) bool {
	if ref == nil {
		return false
	}
	objRefs := o.Obj.GetOwnerReferences()
	for _, r := range objRefs {
		if isReferringSameObject(*ref, r) {
			return false
		}
	}
	objRefs = append(objRefs, *ref)
	o.Obj.SetOwnerReferences(objRefs)
	return true
}

// IsSameAs - return name string
func (o *Object) IsSameAs(a interface{}) bool {
	same := false
	e := a.(*Object)
	oNamespace := o.Obj.GetNamespace()
	oName := o.Obj.GetName()
	oKind := reflect.TypeOf(o.Obj).String()
	if (e.Obj.GetName() == oName) &&
		(e.Obj.GetNamespace() == oNamespace) &&
		(reflect.TypeOf(e.Obj).String() == oKind) {
		same = true
	}
	return same
}

// GetName - return name string
func (o *Object) GetName() string {
	eNamespace := o.Obj.GetNamespace()
	eName := o.Obj.GetName()
	eKind := reflect.TypeOf(o.Obj).String()
	return eNamespace + "/" + eKind + "/" + eName
}

// Observable captures the k8s resource info and selector to fetch child resources
type Observable struct {
	// ObjList refers to the list of resource objects
	ObjList metav1.ListInterface
	// Obj refers to the resource object  can be: sts, service, secret, pvc, ..
	Obj metav1.Object
	// Labels list of labels
	Labels map[string]string
	// Typemeta - needed for go test fake client
	Type metav1.TypeMeta
}

// LocalObjectReference with validation
type LocalObjectReference struct {
	corev1.LocalObjectReference `json:",inline"`
}

// AsItem wraps object as resource item
func (o *Object) AsItem() *resource.Item {
	return &resource.Item{
		Obj:       o,
		Lifecycle: resource.LifecycleManaged,
		Type:      Type,
	}
}

// itemFromReader reads Object from []byte spec
func itemFromReader(name string, b *bufio.Reader, data interface{}, list metav1.ListInterface) (*resource.Item, error) {
	var exdoc bytes.Buffer
	r := yaml.NewYAMLReader(b)
	doc, err := r.Read()
	if err == nil {
		tmpl, e := template.New("tmpl").Parse(string(doc))
		err = e
		if err == nil {
			err = tmpl.Execute(&exdoc, data)
			if err == nil {
				d := scheme.Codecs.UniversalDeserializer()
				obj, _, e := d.Decode(exdoc.Bytes(), nil, nil)
				err = e
				if err == nil {
					return &resource.Item{
						Type:      Type,
						Lifecycle: resource.LifecycleManaged,
						Obj: &Object{
							Obj:     obj.DeepCopyObject().(metav1.Object),
							ObjList: list,
						},
					}, nil
				}
			}
		}
	}

	return nil, errors.New(name + ":" + err.Error())
}

// ItemFromString populates Object from string spec
func ItemFromString(name, spec string, values interface{}, list metav1.ListInterface) (*resource.Item, error) {
	return itemFromReader(name, bufio.NewReader(strings.NewReader(spec)), values, list)
}

// ItemFromFile populates Object from file
func ItemFromFile(path string, values interface{}, list metav1.ListInterface) (*resource.Item, error) {
	f, err := os.Open(path)
	if err == nil {
		return itemFromReader(path, bufio.NewReader(f), values, list)
	}
	return nil, err
}

// NewObservable returns an observable object
func NewObservable(list metav1.ListInterface, labels map[string]string) resource.Observable {
	return resource.Observable{
		Type: Type,
		Obj: Observable{
			ObjList: list,
			Labels:  labels,
		},
	}
}

// ObservablesFromObjects returns ObservablesFromObjects
func (rm *RsrcManager) ObservablesFromObjects(bag *resource.Bag, labels map[string]string) []resource.Observable {
	var gk schema.GroupKind
	var observables []resource.Observable
	gkmap := map[schema.GroupKind]struct{}{}
	for _, item := range bag.Items() {
		if item.Type != Type {
			continue
		}
		obj, ok := item.Obj.(*Object)
		if !ok {
			continue
		}
		if obj.ObjList != nil {
			ro := obj.Obj.(runtime.Object)
			kinds, _, err := rm.scheme.ObjectKinds(ro)
			if err == nil {
				// Expect only 1 kind.  If there is more than one kind this is probably an edge case such as ListOptions.
				if len(kinds) != 1 {
					err = fmt.Errorf("Expected exactly 1 kind for Object %T, but found %s kinds", ro, kinds)

				}
			}
			// Cache the Group and Kind for the OwnerType
			if err == nil {
				gk = schema.GroupKind{Group: kinds[0].Group, Kind: kinds[0].Kind}
			} else {
				gk = ro.GetObjectKind().GroupVersionKind().GroupKind()
			}
			if _, ok := gkmap[gk]; !ok {
				gkmap[gk] = struct{}{}
				observable := resource.Observable{
					Type: Type,
					Obj: Observable{
						ObjList: obj.ObjList,
						Labels:  labels,
					},
				}
				observables = append(observables, observable)
			}
		} else {
			observable := resource.Observable{
				Type: Type,
				Obj: Observable{
					Obj: obj.Obj,
				},
			}
			observables = append(observables, observable)
		}
	}
	return observables
}

// Validate validates the LocalObjectReference
func (s *LocalObjectReference) Validate(fp *field.Path, sfield string, errs field.ErrorList, required bool) field.ErrorList {
	fp = fp.Child(sfield)
	if s == nil {
		if required {
			errs = append(errs, field.Required(fp, "Required "+sfield+" missing"))
		}
		return errs
	}

	if s.Name == "" {
		errs = append(errs, field.Required(fp.Child("name"), "name is required"))
	}
	return errs
}

// FilterObservable - remove from observable
func FilterObservable(observable []resource.Observable, list metav1.ListInterface) []resource.Observable {
	var filtered []resource.Observable
	ltype := reflect.TypeOf(list).String()
	for i := range observable {
		otype := ""
		o, ok := observable[i].Obj.(Observable)
		if ok {
			otype = reflect.TypeOf(o.ObjList).String()
		}
		if ltype != otype {
			filtered = append(filtered, observable[i])
		}
	}
	return filtered

}

// ReferredItem returns a reffered object
func ReferredItem(obj metav1.Object, name, namespace string) resource.Item {
	obj.SetName(name)
	obj.SetNamespace(namespace)
	return resource.Item{
		Lifecycle: resource.LifecycleReferred,
		Type:      Type,
		Obj:       &Object{Obj: obj},
	}
}

// GetItem returns an item which matched the kind and name
func GetItem(b *resource.Bag, inobj metav1.Object, name, namespace string) metav1.Object {
	inobj.SetName(name)
	inobj.SetNamespace(namespace)
	for _, item := range b.ByType(Type) {
		obj := item.Obj.(*Object)
		otype := reflect.TypeOf(obj.Obj).String()
		intype := reflect.TypeOf(inobj).String()
		if otype == intype && obj.Obj.GetName() == inobj.GetName() && obj.Obj.GetNamespace() == inobj.GetNamespace() {
			return obj.Obj
		}
	}
	return nil
}

// Remove returns an item which matched the kind and name
func Remove(b *resource.Bag, inobj metav1.Object) {
	for i, item := range b.Items() {
		if item.Type == Type {
			obj := item.Obj.(*Object)
			otype := reflect.TypeOf(obj.Obj).String()
			intype := reflect.TypeOf(inobj).String()
			if otype == intype && obj.Obj.GetName() == inobj.GetName() && obj.Obj.GetNamespace() == inobj.GetNamespace() {
				b.DeleteAt(i)
				break
			}
		}
	}
}

// Objs get items from the Object bag
func Objs(b *resource.Bag) []metav1.Object {
	var objs []metav1.Object
	for _, item := range b.ByType(Type) {
		o := item.Obj.(*Object)
		objs = append(objs, o.Obj)
	}
	return objs
}

// CopyMutatedSpecFields - copy known mutated fields from observed to expected
func CopyMutatedSpecFields(to *resource.Item, from *resource.Item) {
	e := to.Obj.(*Object)
	o := from.Obj.(*Object)
	e.Obj.SetOwnerReferences(o.Obj.GetOwnerReferences())
	e.Obj.SetResourceVersion(o.Obj.GetResourceVersion())
	// TODO
	switch e.Obj.(type) {
	case *corev1.Service:
		e.Obj.(*corev1.Service).Spec.ClusterIP = o.Obj.(*corev1.Service).Spec.ClusterIP
	case *policyv1.PodDisruptionBudget:
		//e.Obj.SetResourceVersion(o.Obj.GetResourceVersion())
	case *corev1.PersistentVolumeClaim:
		e.Obj.(*corev1.PersistentVolumeClaim).Spec.StorageClassName = o.Obj.(*corev1.PersistentVolumeClaim).Spec.StorageClassName
	}
}

// SpecDiffers - check if the spec part differs
func (rm *RsrcManager) SpecDiffers(expected, observed *resource.Item) bool {
	e := expected.Obj.(*Object)
	o := observed.Obj.(*Object)

	// Not all k8s objects have Spec
	// example ConfigMap
	// TODO strategic merge patch diff in generic controller loop

	CopyMutatedSpecFields(expected, observed)

	espec := reflect.Indirect(reflect.ValueOf(e.Obj)).FieldByName("Spec")
	ospec := reflect.Indirect(reflect.ValueOf(o.Obj)).FieldByName("Spec")
	if !espec.IsValid() {
		// handling ConfigMap
		espec = reflect.Indirect(reflect.ValueOf(e.Obj)).FieldByName("Data")
		ospec = reflect.Indirect(reflect.ValueOf(o.Obj)).FieldByName("Data")
	}
	if espec.IsValid() && ospec.IsValid() {
		if reflect.DeepEqual(espec.Interface(), ospec.Interface()) {
			return false
		}
	}
	return true
}

// Observe - get resources
func (rm *RsrcManager) Observe(observables ...resource.Observable) (*resource.Bag, error) {
	var returnval *resource.Bag = new(resource.Bag)
	var err error
	for _, item := range observables {
		var resources []resource.Item
		obs, ok := item.Obj.(Observable)
		if !ok {
			continue
		}
		if obs.Labels != nil {
			//log.Printf("   >>>list: %s labels:[%v]", reflect.TypeOf(obs.ObjList).String(), obs.Labels)
			opts := client.MatchingLabels(obs.Labels)
			opts.Raw = &metav1.ListOptions{TypeMeta: obs.Type}
			err = rm.client.List(context.TODO(), opts, obs.ObjList.(runtime.Object))
			if err == nil {
				items, err := meta.ExtractList(obs.ObjList.(runtime.Object))
				if err == nil {
					for _, item := range items {
						resources = append(resources, resource.Item{Type: Type, Obj: &Object{Obj: item.(metav1.Object)}})
					}
				}
				/*
					//items := reflect.Indirect(reflect.ValueOf(obs.ObjList)).FieldByName("Items")
					for i := 0; i < items.Len(); i++ {
						o := items.Index(i)
						resources = append(resources, resource.Object{Obj: o.Addr().Interface().(metav1.Object)})
					}
				*/
			}
		} else {
			// check typecasting ?
			// TODO check obj := obs.Obj.(metav1.Object)
			var obj metav1.Object = obs.Obj.(metav1.Object)
			name := obj.GetName()
			namespace := obj.GetNamespace()
			otype := reflect.TypeOf(obj).String()
			err = rm.client.Get(context.TODO(),
				types.NamespacedName{Name: name, Namespace: namespace},
				obs.Obj.(runtime.Object))
			if err == nil {
				log.Printf("   >>get: %s", otype+"/"+namespace+"/"+name)
				resources = append(resources, resource.Item{Type: Type, Obj: &Object{Obj: obs.Obj}})
			} else {
				log.Printf("   >>>ERR get: %s", otype+"/"+namespace+"/"+name)
			}
		}
		if err != nil {
			return nil, err
		}
		for _, resource := range resources {
			returnval.Add(resource)
		}
	}
	return returnval, nil
}

// Update - Generic client update
func (rm *RsrcManager) Update(item resource.Item) error {
	return rm.client.Update(context.TODO(), item.Obj.(*Object).Obj.(runtime.Object).DeepCopyObject())

}

// Create - Generic client create
func (rm *RsrcManager) Create(item resource.Item) error {
	return rm.client.Create(context.TODO(), item.Obj.(*Object).Obj.(runtime.Object))
}

// Delete - Generic client delete
func (rm *RsrcManager) Delete(item resource.Item) error {
	return rm.client.Delete(context.TODO(), item.Obj.(*Object).Obj.(runtime.Object))
}

// Get a specific k8s obj
func Get(rm manager.Manager, nn types.NamespacedName, o runtime.Object) error {
	krm := rm.(*RsrcManager)
	return krm.client.Get(context.TODO(), nn, o)
}
