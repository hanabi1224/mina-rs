#! /usr/bin/sh

SCCACHE_VERSION=v0.2.15
SCCACHE_DIST=sccache-$SCCACHE_VERSION-x86_64-unknown-linux-musl

cd /tmp
wget https://github.com/mozilla/sccache/releases/download/$SCCACHE_VERSION/$SCCACHE_DIST.tar.gz
tar -xvf $SCCACHE_DIST.tar.gz
sudo chmod +x $PWD/$SCCACHE_DIST/sccache
sudo ln -sf $PWD/$SCCACHE_DIST/sccache /usr/bin/sccache
sccache -s
sccache --version
