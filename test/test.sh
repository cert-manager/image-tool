#!/usr/bin/env bash

# Copyright 2022 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

script_dir=$(dirname "$(realpath "$0")")

pushd "${script_dir}" > /dev/null
cd ..

mkdir -p _bin/test

if [ ! -e _bin/test/test-oci.tar ]; then
    # Setup a buildx builder
    cleanup() {
        docker buildx rm testBuilder || true
    }
    trap cleanup EXIT
    docker buildx create --name testBuilder --driver docker-container

    # Create a test OCI tarball
    docker build \
        --builder testBuilder \
        --provenance false \
        --platform linux/amd64,linux/arm64 \
        --output=type=oci,dest=_bin/test/test-oci.tar \
        -f ./test/test.Dockerfile \
        ./test/
fi

# Build the image-tool
go build -o _bin/test/image-tool .
_bin/test/image-tool convert-from-oci-tar _bin/test/test-oci.tar _bin/test/test-oci

_bin/test/image-tool list-digests _bin/test/test-oci

extract_docker() {
    rm -rf _bin/test/test-docker
    mkdir -p _bin/test/test-docker
    tar -xf _bin/test/test-docker.tar -C _bin/test/test-docker
}

_bin/test/image-tool convert-to-docker-tar _bin/test/test-oci _bin/test/test-docker.tar testimage:test-tag
extract_docker
if [ "$(cat _bin/test/test-docker/manifest.json | jq -r '.[].RepoTags[]' | grep testimage:test-tag)" != "testimage:test-tag" ]; then
    echo "❌ Expected testimage:test-tag to be in the manifest.json"
    exit 1
else
    echo "✅︎ Found testimage:test-tag as expected"
fi

_bin/test/image-tool tag-docker-tar _bin/test/test-docker.tar testimage:new-test-tag
extract_docker
if [ "$(cat _bin/test/test-docker/manifest.json | jq -r '.[].RepoTags[]' | grep testimage:new-test-tag)" != "testimage:new-test-tag" ]; then
    echo "❌ Expected testimage:new-test-tag to be in the manifest.json"
    exit 1
else
    echo "✅︎ Found testimage:new-test-tag as expected"
fi

find_labels() {
    # loop over all files in _bin/test/test-oci/blobs/sha256
    # skip files with invalid json or that are lareger than 2MB
    # and try to find all json files with a rootfs key
    for file in _bin/test/test-oci/blobs/sha256/*; do
        if [[ ! -f "$file" ]]; then
            continue
        fi

        # Check if the file is too large
        twoMB=$((2 * 1024 * 1024))
        if [[ $(stat -c%s "$file") -gt $twoMB ]]; then
            continue
        fi

        # Check if the file is a valid JSON file
        if ! jq empty "$file" > /dev/null 2>&1; then
            continue
        fi

        # Check if the file has a rootfs key
        if ! jq -e 'has("rootfs")' "$file" > /dev/null; then
            continue
        fi

        # Return the labels, possibly missing
        cat "$file" | jq -r '.config | select(.Labels != null) | .Labels | to_entries[] | "\(.key)=\(.value)"'
    done
}

labels=$(find_labels)
# $labels contains two duplicate entries (one for each architecture)
# we just check that the labelKey=labelValue is present
if [ "$(echo "$labels" | head -1)" != "labelKey=labelValue" ]; then
    echo "❌ Expected labelKey=labelValue to be in the labels"
    exit 1
else
    echo "✅︎ Found labelKey=labelValue as expected"
fi
_bin/test/image-tool reset-labels-and-annotations _bin/test/test-oci
labels=$(find_labels)
if [ "$labels" != "" ]; then
    echo "❌ Expected no labels"
    exit 1
else
    echo "✅︎ Found no labels as expected"
fi

popd
