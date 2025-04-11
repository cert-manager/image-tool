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
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/spf13/cobra"

	"github.com/cert-manager/image-tool/pkg"
)

var CommandAppendLayers = cobra.Command{
	Use:   "append-layers oci-layout-path [path-to-tarball...]",
	Short: "Appends a tarball or directory to every image in an OCI index",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		oci := args[0]
		extra := args[1:]

		if len(extra) == 0 {
			return
		}

		{
			path, err := layout.FromPath(oci)
			must("could not load oci directory", err)

			index, err := path.ImageIndex()
			must("could not load oci image index", err)

			layers := []untypedLayer{}
			for _, path := range extra {
				layers = append(layers, newUntypedLayerFromPath(path))
			}

			index, err = pkg.MutateOCITree(
				index,
				func(index v1.ImageIndex) v1.ImageIndex {
					return index
				},
				func(img v1.Image) v1.Image {
					imgMediaType, err := img.MediaType()
					must("could not get image media type", err)

					layerType := types.DockerLayer
					if imgMediaType == types.OCIManifestSchema1 {
						layerType = types.OCILayer
					}

					for _, untypedLayer := range layers {
						layer, err := untypedLayer.ToLayer(layerType)
						must("could not load image layer", err)

						img, err = mutate.AppendLayers(img, layer)
						must("could not append layer", err)
					}

					return img
				},
				func(descriptor v1.Descriptor) v1.Descriptor {
					return descriptor
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

type untypedLayer struct {
	tarball tarball.Opener
}

func newUntypedLayer(tarball tarball.Opener) untypedLayer {
	return untypedLayer{tarball: tarball}
}

func newUntypedLayerFromPath(path string) untypedLayer {
	stat, err := os.Stat(path)
	must("could not open directory or tarball", err)

	var layer untypedLayer
	if stat.IsDir() {
		var buf bytes.Buffer

		tw := tar.NewWriter(&buf)

		_ = filepath.Walk(path, func(target string, info fs.FileInfo, err error) error {
			must("walk error", err)

			header, err := tar.FileInfoHeader(info, info.Name())
			must("could not create tar header", err)

			name, err := filepath.Rel(path, target)
			must("could not build relative path", err)

			// Write simplified header, this removes all fields that would cause
			// the build to be non-reproducible (like modtime for example)
			err = tw.WriteHeader(&tar.Header{
				Typeflag: header.Typeflag,
				Name:     name,
				Mode:     header.Mode,
				Linkname: header.Linkname,
				Size:     header.Size,
			})

			must("could not write tar header", err)

			if !info.IsDir() {
				file, err := os.Open(target)
				must("could not write tar contents", err)

				defer file.Close()

				_, err = io.Copy(tw, file)
				must("could not write tar contents", err)
			}

			return nil
		})

		tw.Close()

		byts := buf.Bytes()

		layer = newUntypedLayer(
			func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(byts)), nil
			},
		)
	} else {
		layer = newUntypedLayer(
			func() (io.ReadCloser, error) {
				return os.Open(path)
			},
		)
	}

	return layer
}

func (ul untypedLayer) ToLayer(mediaType types.MediaType) (v1.Layer, error) {
	return tarball.LayerFromOpener(ul.tarball, tarball.WithMediaType(mediaType))
}
