#!/bin/sh

TAG=$1
[ -z "$TAG" ] && echo "Missing tag name (vX.Y.Z) as first arg" && exit 1

./build.sh
curl -Lo hub.tgz https://github.com/github/hub/releases/download/v2.11.2/hub-darwin-amd64-2.11.2.tgz
tar xzf hub.tgz && rm hub.tgz
./hub-darwin-amd64-2.11.2/bin/hub release create -a release/k8s-ingress-hosts-darwin-amd64 -a release/k8s-ingress-hosts-darwin-amd64.sha256 -a release/k8s-ingress-hosts-linux-amd64 -a release/k8s-ingress-hosts-linux-amd64.sha256 $TAG
rm -r hub-darwin-amd64-2.11.2*
