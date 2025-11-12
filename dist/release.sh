#!/bin/sh -e
cd "$(dirname -- "$0")/.."
VERSION="${HAKUREI_VERSION:-untagged}"
pname="hakurei-${VERSION}"
out="dist/${pname}"

mkdir -p "${out}"
cp -v "README.md" "dist/hsurc.default" "dist/install.sh" "${out}"
cp -rv "dist/comp" "${out}"

go generate ./...
go build -trimpath -v -o "${out}/bin/" -ldflags "-s -w -buildid= -extldflags '-static'
  -X hakurei.app/internal/info.buildVersion=${VERSION}
  -X hakurei.app/internal/info.hakureiPath=/usr/bin/hakurei
  -X hakurei.app/internal/info.hsuPath=/usr/bin/hsu
  -X main.hakureiPath=/usr/bin/hakurei" ./...

rm -f "./${out}.tar.gz" && tar -C dist -czf "${out}.tar.gz" "${pname}"
rm -rf "./${out}"
(cd dist && sha512sum "${pname}.tar.gz" > "${pname}.tar.gz.sha512")
