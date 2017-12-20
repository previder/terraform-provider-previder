#!/bin/bash -e
NAME="terraform-provider-previder"
PACKAGE="github.com/previder/$NAME"
BUILD_OS=${BUILD_OS:-darwin linux windows freebsd}
BUILD_ARCH=${BUILD_ARCH:-386 amd64}
ldflags="-s -w"
for os in ${BUILD_OS}; do
  export GOOS="${os}"
  for arch in ${BUILD_ARCH}; do
    export GOARCH="${arch}"
    out="${NAME}_${os}_${arch}"
    if [ "${os}" == "windows" ]; then
      out="${out}.exe"
    fi
    set -x
    go build \
      -o="${out}" \
      -pkgdir="./_pkg" \
      -compiler='gc' \
      -ldflags="${ldflags}" \
      $PACKAGE &
    set +x
  done
done
wait
sha256sum ${NAME}_* >SHA256SUMS
