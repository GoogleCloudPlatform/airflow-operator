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

package resource

// Add adds to the Object bag
func (b *Bag) Add(items ...Item) {
	b.items = append(b.items, items...)
}

// Merge a Bag with this one
func (b *Bag) Merge(from *Bag) {
	b.Add(from.Items()...)
}

// Items get items from the Object bag
func (b *Bag) Items() []Item {
	return b.items
}

// ByType get items from the Object bag
func (b *Bag) ByType(t string) []Item {
	var items []Item
	for _, item := range b.items {
		if item.Type == t {
			items = append(items, item)
		}
	}
	return items
}

// DeleteAt object at index
func (b *Bag) DeleteAt(index int) {
	b.items[index] = b.items[len(b.items)-1]
	b.items = b.items[:len(b.items)-1]
}

// ItemAt item at index
func (b *Bag) ItemAt(index int) *Item {
	return &b.items[index]
}
