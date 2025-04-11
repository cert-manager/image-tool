# Copyright 2025 The cert-manager Authors.
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

repo_name := github.com/cert-manager/image-tool

exe_build_names := image-tool
gorelease_file := .goreleaser.yml

go_image-tool_main_dir := .
go_image-tool_mod_dir := .
go_image-tool_ldflags := \
	-X $(repo_name)/internal/version.AppVersion=$(VERSION) \
	-X $(repo_name)/internal/version.GitCommit=$(GITCOMMIT)

golangci_lint_config := .golangci.yaml
