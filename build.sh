#!/bin/bash

set -euo pipefail


git clone --branch bifrost-image --depth 1 https://github.com/achilleas-k/images.git
cd images
go build ./cmd/osbuild-deploy-container


git clone --branch osbuild-deploy-container-test --depth 1 https://github.com/mvo5/osbuild
cd osbuild
make rpm
