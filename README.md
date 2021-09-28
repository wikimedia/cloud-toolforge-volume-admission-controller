# volume-admission-controller

Automatically mount volumes for [Toolforge](https://toolforge.org) pods.

It is based on [ingress-admission-controller](https://gerrit.wikimedia.org/r/plugins/gitiles/cloud/toolforge/ingress-admission-controller).

## Deploying locally on Minikube

First, build the image inside Minikube:

```
eval $(minikube docker-env)
docker build -t volume-admission:latest .
```

Then, create a config file to contain your Minikube cluster CA and
create a certificate the webhook will use when listening for requests:

```
./deployment/ca-bundle.sh
./deployment/get-cert.sh
```

Then just apply the manifests:

```
kubectl apply -k deployment/deploys/local
```

### Updating

To apply your new changes, repeat the build step and then run the
following command to re-create the running containers:

```
kubectl delete pod --all -n volume-admission
```

## Deploying on Toolforge

See instructions on [Wikitech](https://wikitech.wikimedia.org/wiki/Portal:Toolforge/Admin/Kubernetes/Deploying#volume_admission).

