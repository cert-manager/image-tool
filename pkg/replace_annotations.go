/*
Copyright 2025 The cert-manager Authors.

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

package pkg

import (
	"encoding/json"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

func ReplaceImageAnnotations(f v1.Image, annotations map[string]string) v1.Image {
	return imageAnnotationsReplacer{
		Image:       f,
		annotations: annotations,
	}
}

func ReplaceImageIndexAnnotations(f v1.ImageIndex, annotations map[string]string) v1.ImageIndex {
	return indexAnnotationsReplacer{
		embededImageIndex: f,
		annotations:       annotations,
	}
}

func replaceAnnotations(f partial.WithRawManifest, annotations map[string]string) ([]byte, error) {
	b, err := f.RawManifest()
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	if len(annotations) == 0 {
		delete(m, "annotations")
	} else {
		m["annotations"] = annotations
	}

	return json.Marshal(m)
}

type imageAnnotationsReplacer struct {
	v1.Image
	annotations map[string]string
}

func (a imageAnnotationsReplacer) RawManifest() ([]byte, error) {
	return replaceAnnotations(a.Image, a.annotations)
}

func (a imageAnnotationsReplacer) Digest() (v1.Hash, error) {
	return partial.Digest(a)
}

func (a imageAnnotationsReplacer) Manifest() (*v1.Manifest, error) {
	return partial.Manifest(a)
}

type embededImageIndex = v1.ImageIndex

type indexAnnotationsReplacer struct {
	embededImageIndex
	annotations map[string]string
}

func (a indexAnnotationsReplacer) RawManifest() ([]byte, error) {
	return replaceAnnotations(a.embededImageIndex, a.annotations)
}

func (a indexAnnotationsReplacer) Digest() (v1.Hash, error) {
	return partial.Digest(a)
}

func (a indexAnnotationsReplacer) Manifest() (*v1.Manifest, error) {
	return partial.Manifest(a)
}
