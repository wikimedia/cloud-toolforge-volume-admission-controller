#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 |tr -d '\n')

cat > $(dirname $0)/deploys/local/webhook.yaml <<EOF
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: volume-admission
webhooks:
  - name: volume-admission.tools.wmcloud.org
    clientConfig:
      caBundle: ${CA_BUNDLE}
EOF
