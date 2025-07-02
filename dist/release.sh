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
  -X git.gensokyo.uk/security/hakurei/internal.version=${VERSION}
  -X git.gensokyo.uk/security/hakurei/internal.hmain=/usr/bin/hakurei
  -X git.gensokyo.uk/security/hakurei/internal.hsu=/usr/bin/hsu
  -X main.hmain=/usr/bin/hakurei" ./...

rm -f "./${out}.tar.gz" && tar -C dist -czf "${out}.tar.gz" "${pname}"
rm -rf "./${out}"
(cd dist && sha512sum "${pname}.tar.gz" > "${pname}.tar.gz.sha512")