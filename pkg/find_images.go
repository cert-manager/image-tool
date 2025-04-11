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
)

func FindImagesInOCITree(index v1.ImageIndex, matcher match.Matcher) ([]v1.Image, error) {
	manifest, err := index.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("could not load oci index manifest: %w", err)
	}

	var images []v1.Image
	for _, descriptor := range manifest.Manifests {
		switch {
		case descriptor.MediaType.IsImage():
			// If the platform is not part of the index manifest, attempt to
			// load it from the image config
			if descriptor.Platform == nil {
				img, err := index.Image(descriptor.Digest)
				if err != nil {
					return nil, fmt.Errorf("could not load image: %w", err)
				}

				cfg, err := img.ConfigFile()
				if err != nil {
					return nil, fmt.Errorf("could not load image config: %w", err)
				}

				descriptor.Platform = cfg.Platform()
			}

			if matcher(descriptor) {
				img, err := index.Image(descriptor.Digest)
				if err != nil {
					return nil, fmt.Errorf("could not load image: %w", err)
				}

				images = append(images, img)
			}

		case descriptor.MediaType.IsIndex():
			idx, err := index.ImageIndex(descriptor.Digest)
			if err != nil {
				return nil, fmt.Errorf("could not load image index: %w", err)
			}

			extraImages, err := FindImagesInOCITree(idx, matcher)
			if err != nil {
				return nil, err
			}

			images = append(images, extraImages...)
		}
	}

	return images, nil
}
