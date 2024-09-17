#!/bin/bash

# Variables
CHART_NAME="camphouse"
RELEASE_NAME="camphouse"
HELM_VALUES_DIR="./camphouse"
# no "stag" "prod" for now
ENVIRONMENTS=("dev")
VERSION="next"

# Usage function
usage() {
    echo "Usage: $0 [--version <version>] <environment1> [environment2 ...]"
    exit 1
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    
    case $key in
        --version)
            VERSION="$2"
            shift # past argument
            shift # past value
            ;;
        -h|--help)
            usage
            ;;
        *)
            # Assume this is an environment argument
            if [[ ! " ${ENVIRONMENTS[@]} " =~ " $1 " ]]; then
                echo "Invalid environment: $1. Please specify one of the following environments: ${ENVIRONMENTS[*]}"
                exit 1
            fi
            ENVIRONMENTS_TO_DEPLOY+=("$1")
            shift # past argument
            ;;
    esac
done

# Check if any environments are provided
if [ ${#ENVIRONMENTS_TO_DEPLOY[@]} -eq 0 ]; then
    echo "No environments provided. Please specify at least one environment (e.g., dev)."
    usage
fi

echo "Deploying version ${VERSION} of ${CHART_NAME} to the following environments: ${ENVIRONMENTS_TO_DEPLOY[*]}"

# Loop through each provided environment and deploy the Helm chart
for ENVIRONMENT in "${ENVIRONMENTS_TO_DEPLOY[@]}"; do
    echo "Deploying ${CHART_NAME} to ${ENVIRONMENT}..."

    # Check if the corresponding values file exists
    VALUES_FILE="${HELM_VALUES_DIR}/values-${ENVIRONMENT}.yaml"
    if [ ! -f "$VALUES_FILE" ]; then
        echo "Values file for environment '${ENVIRONMENT}' not found: ${VALUES_FILE}"
        exit 1
    fi

    NAMESPACE="${RELEASE_NAME}-${ENVIRONMENT}"

    # Create the namespace if it doesn't exist
    kubectl get namespace "$NAMESPACE" || kubectl create namespace "$NAMESPACE"

    # Deploy the Helm chart
    helm upgrade --install \
        "${NAMESPACE}" \
        "${CHART_NAME}" \
        --namespace "$NAMESPACE" \
        --values "$VALUES_FILE" \
        --set "version=${VERSION}"

    echo "Deployment to ${ENVIRONMENT} completed."
done

echo "All deployments completed successfully."
