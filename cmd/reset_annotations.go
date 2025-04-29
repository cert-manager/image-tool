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

package cmd

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/spf13/cobra"

	"github.com/cert-manager/image-tool/pkg"
)

var CommandResetLabelsAndAnnotations = cobra.Command{
	Use:   "reset-labels-and-annotations oci-layout-path",
	Short: "Removes all labels and annotations from OCI indices, images and descriptors in a OCI layout directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		oci := args[0]

		{
			path, err := layout.FromPath(oci)
			must("could not load oci directory", err)

			index, err := path.ImageIndex()
			must("could not load oci image index", err)

			index, err = pkg.MutateOCITree(
				index,
				func(index v1.ImageIndex) (v1.ImageIndex, error) {
					return pkg.ReplaceImageIndexAnnotations(index, map[string]string{}), nil
				}, func(image v1.Image) (v1.Image, error) {
					configFile, err := image.ConfigFile()
					if err != nil {
						return nil, fmt.Errorf("could not parse config file: %w", err)
					}

					configFile.Config.Labels = map[string]string{}

					image, err = mutate.ConfigFile(image, configFile)
					if err != nil {
						return nil, fmt.Errorf("could not replace config file: %w", err)
					}

					return pkg.ReplaceImageAnnotations(image, map[string]string{}), nil
				}, func(descriptor v1.Descriptor) (v1.Descriptor, error) {
					descriptor.Annotations = map[string]string{}
					return descriptor, nil
				},
			)
			must("could not modify oci tree", err)

			_, err = layout.Write(oci, index)
			must("could not write image", err)
		}

		{
			path, err := layout.FromPath(oci)
			must("could not load oci directory", err)

			hashesToRemove, err := path.GarbageCollect()
			must("could not garbage collect oci image", err)

			for _, hash := range hashesToRemove {
				err := path.RemoveBlob(hash)
				must("could not remove blob", err)
			}
		}
	},
}
