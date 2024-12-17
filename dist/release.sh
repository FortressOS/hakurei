#!/bin/sh -e
cd "$(dirname -- "$0")/.."
VERSION="${FORTIFY_VERSION:-untagged}"
pname="fortify-${VERSION}"
out="dist/${pname}"

mkdir -p "${out}"
cp "README.md" "dist/fsurc.default" "dist/install.sh" "${out}"

go build -v -o "${out}/bin/" -ldflags "-s -w
  -X git.ophivana.moe/security/fortify/internal.Version=${VERSION}
  -X git.ophivana.moe/security/fortify/internal.Fsu=/usr/bin/fsu
  -X git.ophivana.moe/security/fortify/internal.Finit=/usr/libexec/fortify/finit
  -X main.Fmain=/usr/bin/fortify
  -X main.Fshim=/usr/libexec/fortify/fshim" ./...

rm -f "./${out}.tar.gz" && tar -C dist -czf "${out}.tar.gz" "${pname}"
rm -rf "./${out}"
sha512sum "${out}.tar.gz" > "${out}.tar.gz.sha512"