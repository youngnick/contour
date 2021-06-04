#! /usr/bin/env bash

set -o pipefail
set -o errexit
set -o nounset

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

mkdir -p $BASEDIR/devcache/
kubectl get secret -n projectcontour contourcert -o json > $BASEDIR/devcache/contourcert.json

cat $BASEDIR/devcache/contourcert.json | jq -r '.data."ca.crt"' | base64 -d > $BASEDIR/devcache/ca.crt
cat $BASEDIR/devcache/contourcert.json | jq -r '.data."tls.crt"' | base64 -d > $BASEDIR/devcache/tls.crt
cat $BASEDIR/devcache/contourcert.json | jq -r '.data."tls.key"' | base64 -d > $BASEDIR/devcache/tls.key

kubectl delete service -n projectcontour contour --ignore-not-found

if [ -z "$DOCKERIP" ]
then
    echo "Can't determine Docker IP, make sure DOCKERIP is set."
    exit 1
fi

envsubst < $BASEDIR/templates/headless-contour-service.yaml | kubectl apply -f -
