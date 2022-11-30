#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail


help() {
    cat <<EOH
    Usage: $0 [OPTIONS] <ENVIRONMENT>

    Options:
      -h        Show this help.
      -b        Also build the container image (locally only).
      -v        Show verbose output.

EOH
}


check_environment() {
    # verify that the proper environment is passed

    if [[ ! -d "deployment/deploys/$environment" ]]; then
        echo "Unknown environment $environment, use one of:"
        ls deployment/deploys/
        exit 1
    fi
}


build_image() {
    # build container image for dev or prod environments

    local environment="${1?No environment passed}"

    if [[ "$environment" == "local" ]]
    then
        # creates the image on minikube's docker daemon
        eval $(minikube docker-env)
        docker build -t volume-admission:latest .
    else
        # Build the container image on the docker-builder host (currently tools-docker-imagebuilder-01.tools.eqiad1.wikimedia.cloud).
        docker build . -f Dockerfile -t "docker-registry.tools.wmflabs.org/volume-admission:latest"
        # Push the image to the internal repo
        docker push "docker-registry.tools.wmflabs.org/volume-admission:latest"
        echo "Successfully built container image for tools/toolsbeta environments with exit code 0. \
        To deploy,log into k8s control node with repository checked out and run './deploy.sh (tools or toolsbeta)'"
    fi
}

deploy_generic() {
    # deploy buildpack-admission-controller image to either dev or prod environments

    local environment="${1?No environment passed}"

    if [[ "$environment" == "local" ]]; then
        deployment/ca-bundle.sh
        deployment/get-cert.sh
    fi
    kubectl apply -k "deployment/deploys/${environment}"
}


main () {
    local do_build="no"

    while getopts "hbv" option; do
        case "${option}" in
        h)
            help
            exit 0
            ;;
        b) do_build="yes";;
        v) set -x;;
        *)
            echo "Wrong option $option"
            help
            exit 1
            ;;
        esac
    done
    shift $((OPTIND-1))

    # default to prod, avoid deploying dev in prod if there's any issue
    local environment="tools"
    if [[ "${1:-}" == "" ]]; then
        if [[ -f /etc/wmcs-project ]]; then
            environment="$(cat /etc/wmcs-project)"
        fi
    else
        environment="${1:-}"
    fi

    check_environment "$environment"

    if [[ "$do_build" == "yes" ]];then
        build_image "$environment"
        if [[ "$environment" != "local" ]];then
            exit 0
        fi
    fi

    deploy_generic "$environment"
}

main "$@"

