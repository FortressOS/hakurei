#!/bin/sh -e
cd "$(dirname -- "$0")/.."
VERSION="${FORTIFY_VERSION:-untagged}"
pname="fortify-${VERSION}"
out="dist/${pname}"

mkdir -p "${out}"
cp -v "README.md" "dist/fsurc.default" "dist/install.sh" "${out}"
cp -rv "comp" "${out}"

go generate ./...
go build -trimpath -v -o "${out}/bin/" -ldflags "-s -w -buildid= -extldflags '-static'
  -X git.gensokyo.uk/security/fortify/internal.Version=${VERSION}
  -X git.gensokyo.uk/security/fortify/internal.Fortify=/usr/bin/fortify
  -X git.gensokyo.uk/security/fortify/internal.Fsu=/usr/bin/fsu
  -X git.gensokyo.uk/security/fortify/internal.Finit=/usr/libexec/fortify/finit
  -X main.Fmain=/usr/bin/fortify
  -X main.Fshim=/usr/libexec/fortify/fshim" ./...

rm -f "./${out}.tar.gz" && tar -C dist -czf "${out}.tar.gz" "${pname}"
rm -rf "./${out}"
(cd dist && sha512sum "${pname}.tar.gz" > "${pname}.tar.gz.sha512")