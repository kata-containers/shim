#!/bin/bash
#
# Copyright (c) 2018 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

set -e

tests_repo="github.com/kata-containers/tests"
tests_repo_dir="${GOPATH}/src/${tests_repo}"

go get -d "$tests_repo" || true
(cd "$tests_repo_dir" && bash ".ci/static-checks.sh")
