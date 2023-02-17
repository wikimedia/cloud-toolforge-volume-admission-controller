#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 |tr -d '\n')

cat > $(dirname $0)/../values/local.yaml <<EOF
image:
  name: volume-admission
  tag: latest
  pullPolicy: Never

webhook:
  caBundle: ${CA_BUNDLE}
EOF
