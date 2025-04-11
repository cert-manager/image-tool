# Image tool

Tool for handling OCI images that implements functionality missing from docker, crane and ko.

## Usage

- `convert-from-oci-tar oci-tarball oci-layout-path` - Reads the OCI layout tarball (=docker build output) and outputs an OCI layout directory (=ko output, crane and image-tool input)
- `append-layers oci-layout-path [path-to-tarball...]` - Appends a tarball or directory to every image in an OCI index
- `convert-to-docker-tar oci-layout-path docker-tarball image-name` - Reads the OCI layout directory and outputs a tarball that is compatible with \"docker load\"
- `list-digests oci-layout-path` - Outputs the digests for images found in the OCI layout directory
- `reset-labels-and-annotations oci-layout-path` - Removes all labels and annotations from OCI indices, images and descriptors in a OCI layout directory
- `tag-docker-tar docker-tarball image-name` - Replaces the image name in the docker tarball (image name should include a tag)
