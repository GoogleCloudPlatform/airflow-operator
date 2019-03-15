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

package reconciler

// NoUpdate - set lifecycle to noupdate
func NoUpdate(o *Object, v interface{}) {
	o.Lifecycle = LifecycleNoUpdate
}

// Merge is used to merge multiple maps into the target map
func (out KVMap) Merge(kvmaps ...KVMap) {
	for _, kvmap := range kvmaps {
		for k, v := range kvmap {
			out[k] = v
		}
	}
}

// ObjectsByType get items from the Object bag
func ObjectsByType(in []Object, t string) []Object {
	var out []Object
	for _, item := range in {
		if item.Type == t {
			out = append(out, item)
		}
	}
	return out
}

// DeleteAt object at index
func DeleteAt(in []Object, index int) []Object {
	var out []Object
	in[index] = in[len(in)-1]
	out = in[:len(in)-1]
	return out
}
