#!/usr/bin/env bash

# Copyright 2017 KubeSphere Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

CRD_OPTIONS="$1"
PKGS="$2"
GENS="$3"
IFS=" " read -r -a PKGS <<< "${PKGS}"
export GOFLAGS=-mod=readonly

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${KUBE_ROOT}" || exit

for PKG in "${PKGS[@]}"; do
  if grep -qw "deepcopy" <<<"${GENS}"; then
    echo "Generating deepcopy for ${PKG}"
    go run ./vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go object:headerFile=./hack/boilerplate.go.txt paths=./api/"${PKG}"
  else
    echo "Generating manifests for ${PKG}"
    go run ./vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go object:headerFile=./hack/boilerplate.go.txt paths=./api/"${PKG}" rbac:roleName=controller-perms "${CRD_OPTIONS}" output:crd:artifacts:config=charts/mutato/crds
  fi
done
