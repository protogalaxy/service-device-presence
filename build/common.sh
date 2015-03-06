#!/bin/bash

# This is free and unencumbered software released into the public domain.
#
# Anyone is free to copy, modify, publish, use, compile, sell, or
# distribute this software, either in source code form or as a compiled
# binary, for any purpose, commercial or non-commercial, and by any
# means.
#
# In jurisdictions that recognize copyright laws, the author or authors
# of this software dedicate any and all copyright interest in the
# software to the public domain. We make this dedication for the benefit
# of the public at large and to the detriment of our heirs and
# successors. We intend this dedication to be an overt act of
# relinquishment in perpetuity of all present and future rights to this
# software under copyright law.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.
#
# For more information, please refer to <http://unlicense.org/>

set -eu -o pipefail

PG_ROOT=$(pwd)
cd $PG_ROOT

source "${PG_ROOT}/build/golang.sh"

RUN_ENV="${RUN_ENV:-local}"

readonly LOCAL_TARGET_ROOT="${PG_ROOT}/target"

readonly DOCKER_TARGET_ROOT="/target"

if [[ "${RUN_ENV}" == "docker" ]]; then
  readonly TARGET_ROOT="${DOCKER_TARGET_ROOT}"
else
  readonly TARGET_ROOT="${LOCAL_TARGET_ROOT}"
fi

readonly TARGET_BIN="${TARGET_ROOT}/bin"
readonly TARGET_TEST="${TARGET_ROOT}/test"
readonly TARGET_TEST_COVERAGE="${TARGET_TEST}/coverage"

readonly PG_BUILD_IMAGE_NAME=protogalaxy-build

function pg::build::verify() {
  echo "+++ Verifying prerequisites ..."

  if [[ -z "$(which docker)" ]]; then
    echo "Can't find docker executable in PATH." >&2
    exit 1
  fi
}

function pg::build::build_image() {
  echo "+++ Builing docker image: ${PG_BUILD_IMAGE_NAME}"
  docker build -t "${PG_BUILD_IMAGE_NAME}" "$PG_ROOT/build/build-image"
}

function pg::build::run_command() {
  local -r cmd=$1

  local -r path="${GOPATH}/${PROJECT_PATH}"

  echo "+++ Running command ..."
  rm -rf cid
  docker run --cidfile=cid \
    -e RUN_ENV=docker \
    -v "$(pwd)":"${path}" \
    -w "${path}" \
    "${PG_BUILD_IMAGE_NAME}" "$cmd"

  local cid
  cid=$(cat cid)

  echo "+++ Copying built files from the build container"
  docker cp $cid:$DOCKER_TARGET_ROOT $PG_ROOT

  echo "+++ Removing the build container"
  docker rm $cid 2> /dev/null || true
  rm -rf cid
}

function pg::build::build_release_image() {
  local -r image_name=$1

  echo "+++ Builing docker image: ${image_name}"
  docker build -t "${image_name}" "$PG_ROOT"
}
