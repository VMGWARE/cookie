#!/bin/bash

# Variables
CHART_NAME="camphouse"
RELEASE_NAME="camphouse"
HELM_VALUES_DIR="./camphouse"
# no "stag" "prod" for now
ENVIRONMENTS=("dev")

# Check if any environments are provided
if [ $# -eq 0 ]; then
  echo "No environments provided. Please specify at least one environment (e.g., dev, stag, prod)."
  exit 1
fi

# Validate the provided environments
for ENVIRONMENT in "$@"; do
    if [[ ! " ${ENVIRONMENTS[@]} " =~ " ${ENVIRONMENT} " ]]; then
        echo "Invalid environment: ${ENVIRONMENT}. Please specify one of the following environments: ${ENVIRONMENTS[*]}"
        exit 1
    fi
done

# Loop through each provided environment and deploy the Helm chart
for ENVIRONMENT in "$@"; do
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

    # Ensure the secrets are created in the namespace: camphouse-secrets

    # Deploy the Helm chart
    helm upgrade --install \
        "${NAMESPACE}" \
        "${CHART_NAME}" \
        --namespace "$NAMESPACE" \
        --values "$VALUES_FILE"

    echo "Deployment to ${ENVIRONMENT} completed."
done

echo "All deployments completed successfully."
