#!/bin/bash

set -e

project=$(cat /etc/wmcs-project || echo "local")

if [ "${project}" == "local" ] ; then
    deployment/ca-bundle.sh
    deployment/get-cert.sh
fi

kubectl apply -k deployment/deploys/${project}
