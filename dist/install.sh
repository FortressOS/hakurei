#!/bin/sh
cd "$(dirname -- "$0")" || exit 1

install -vDm0755 "bin/hakurei" "${HAKUREI_INSTALL_PREFIX}/usr/bin/hakurei"
install -vDm0755 "bin/fpkg" "${HAKUREI_INSTALL_PREFIX}/usr/bin/fpkg"

install -vDm6511 "bin/hsu" "${HAKUREI_INSTALL_PREFIX}/usr/bin/hsu"
if [ ! -f "${HAKUREI_INSTALL_PREFIX}/etc/hsurc" ]; then
    install -vDm0400 "hsurc.default" "${HAKUREI_INSTALL_PREFIX}/etc/hsurc"
fi

install -vDm0644 "comp/_hakurei" "${HAKUREI_INSTALL_PREFIX}/usr/share/zsh/site-functions/_hakurei"