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
	"runtime"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"

	"github.com/cert-manager/image-tool/pkg"
)

var CommandConvertToDockerTar = cobra.Command{
	Use:   "convert-to-docker-tar oci-layout-path docker-tarball image-name",
	Short: "Reads the OCI layout directory and outputs a tarball that is compatible with \"docker load\"",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		output := args[1]
		imageName := args[2]

		ociLayout, err := layout.FromPath(path)
		must("could not load oci directory", err)

		index, err := ociLayout.ImageIndex()
		must("could not load oci image index", err)

		var images []v1.Image
		err = pkg.SearchOCITree(index, nil,
			func(descriptors []*v1.Descriptor, image v1.Image) error {
				var platform *v1.Platform

				for _, desc := range descriptors {
					if desc.Platform != nil {
						platform = desc.Platform
					}
				}

				{
					cfg, err := image.ConfigFile()
					if err != nil {
						return fmt.Errorf("could not load image config: %w", err)
					}
					if imgPlatform := cfg.Platform(); imgPlatform != nil {
						platform = imgPlatform
					}
				}

				if platform != nil && platform.Architecture == runtime.GOARCH {
					images = append(images, image)
				}

				return nil
			},
		)
		must("could not find images", err)

		switch {
		case len(images) == 0:
			fail("no matching images found")
		case len(images) > 1:
			fail("multiple matching images found")
		}

		ref, err := name.ParseReference(imageName)
		must("invalid image name", err)

		err = tarball.WriteToFile(output, ref, images[0])
		must("could not write tarball", err)
	},
}
