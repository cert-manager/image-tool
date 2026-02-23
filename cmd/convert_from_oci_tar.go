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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/spf13/cobra"
)

const maxFileSize = 500 * 1 << 20 // 500 Megabyte = 500 * 1024 * 1024 bytes

var CommandConvertFromOCITar = cobra.Command{
	Use:   "convert-from-oci-tar oci-tarball oci-layout-path",
	Short: "Reads the OCI layout tarball (=docker build output) and outputs an OCI layout directory (=ko output, crane and image-tool input)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		output := args[1]

		{
			err := untar(path, output)
			must("could not untar OCI tarball", err)
		}

		{
			path, err := layout.FromPath(output)
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

func cleanJoin(baseDir, unsafePath string) (string, error) {
	if filepath.IsAbs(unsafePath) {
		return "", fmt.Errorf("absolute paths not allowed: %s", unsafePath)
	}

	cleaned := filepath.Clean(unsafePath)

	if strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) || cleaned == ".." {
		return "", fmt.Errorf("path traversal detected: %s", unsafePath)
	}

	fullPath := filepath.Join(baseDir, cleaned)

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	absTarget, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes base directory: %s", unsafePath)
	}

	return absTarget, nil
}

func untar(src string, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path, err := cleanJoin(dest, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// #nosec G703 -- path validated via cleanJoin to prevent traversal
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// #nosec G703 -- directory path validated via cleanJoin
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}

			if header.Mode < 0 || header.Mode > 0o777 {
				return fmt.Errorf("invalid file mode in tar header: %d", header.Mode)
			}
			perm := os.FileMode(uint32(header.Mode))
			// #nosec G703 -- path validated via cleanJoin to prevent traversal
			outFile, err := os.OpenFile(
				path,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
				perm,
			)
			if err != nil {
				return err
			}

			written, err := io.Copy(outFile, io.LimitReader(tarReader, maxFileSize))
			outFile.Close()
			if err != nil {
				return err
			} else if written == maxFileSize {
				// Prevents G110: Potential DoS vulnerability via decompression bomb
				return fmt.Errorf("tar contained file larger than 500MB")
			}
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("symlinks not allowed in tar archive: %s", header.Name)
		default:
			return fmt.Errorf("unable to untar type: %c in file %s", header.Typeflag, header.Name)
		}
	}
	return nil
}
