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
	"fmt"
	"slices"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type IndexSearchFn func(descriptors []*v1.Descriptor, index v1.ImageIndex) error
type ImageSearchFn func(descriptors []*v1.Descriptor, image v1.Image) error

func SearchOCITree(
	index v1.ImageIndex,
	searchIndexFn IndexSearchFn,
	searchImageFn ImageSearchFn,
) error {
	return searchOCITree(index, nil, searchIndexFn, searchImageFn)
}

func searchOCITree(
	index v1.ImageIndex,
	descriptors []*v1.Descriptor,
	searchIndexFn IndexSearchFn,
	searchImageFn ImageSearchFn,
) error {
	manifest, err := index.IndexManifest()
	if err != nil {
		return fmt.Errorf("could not load oci image manifest: %w", err)
	}

	for _, descriptor := range manifest.Manifests {
		childDescriptors := append(slices.Clip(descriptors), &descriptor)
		switch {
		case descriptor.MediaType.IsImage():
			childImg, err := index.Image(descriptor.Digest)
			if err != nil {
				return fmt.Errorf("could not load oci image from digest: %w", err)
			}

			if searchImageFn != nil {
				if err := searchImageFn(childDescriptors, childImg); err != nil {
					return fmt.Errorf("could not mutate oci image: %w", err)
				}
			}
		case descriptor.MediaType.IsIndex():
			childIndex, err := index.ImageIndex(descriptor.Digest)
			if err != nil {
				return fmt.Errorf("could not load oci image index from digest: %w", err)
			}

			if err := searchOCITree(childIndex, childDescriptors, searchIndexFn, searchImageFn); err != nil {
				return err
			}
		default:
			continue
		}
	}

	if searchIndexFn != nil {
		if err := searchIndexFn(descriptors, index); err != nil {
			return fmt.Errorf("could not mutate index: %w", err)
		}
	}

	return nil
}
