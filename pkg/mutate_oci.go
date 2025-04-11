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

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

type IndexMutateFn func(index v1.ImageIndex) v1.ImageIndex
type ImageMutateFn func(image v1.Image) v1.Image
type DescriptorMutateFn func(descriptor v1.Descriptor) v1.Descriptor

func MutateOCITree(
	index v1.ImageIndex,
	mutIndexFn IndexMutateFn,
	mutImageFn ImageMutateFn,
	mutDescriptorFn DescriptorMutateFn,
) (v1.ImageIndex, error) {
	manifest, err := index.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("could not load oci image manifest: %w", err)
	}

	for _, descriptor := range manifest.Manifests {
		var child mutate.Appendable

		switch {
		case descriptor.MediaType.IsImage():
			childImg, err := index.Image(descriptor.Digest)
			if err != nil {
				return nil, fmt.Errorf("could not load oci image from digest: %w", err)
			}

			childImg = mutImageFn(childImg)

			child = childImg
		case descriptor.MediaType.IsIndex():
			childIndex, err := index.ImageIndex(descriptor.Digest)
			if err != nil {
				return nil, fmt.Errorf("could not load oci image index from digest: %w", err)
			}

			childIndex, err = MutateOCITree(childIndex, mutIndexFn, mutImageFn, mutDescriptorFn)
			if err != nil {
				return nil, err
			}

			child = childIndex
		default:
			continue
		}

		digest, err := child.Digest()
		if err != nil {
			return nil, fmt.Errorf("could not get image digest: %w", err)
		}

		size, err := child.Size()
		if err != nil {
			return nil, fmt.Errorf("could not get image size: %w", err)
		}

		// Remove descriptor from index and re-add descriptor
		index = mutate.RemoveManifests(index, match.Digests(descriptor.Digest))
		descriptor.Digest = digest
		descriptor.Size = size
		index = mutate.AppendManifests(index, mutate.IndexAddendum{
			Add:        child,
			Descriptor: mutDescriptorFn(descriptor),
		})
	}

	return mutIndexFn(index), nil
}
