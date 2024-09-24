#!/usr/bin/env bash

# Define the base64-encoded auth strings for Docker and Harbor registries
docker_auth=$(printf "%s:%s" "${DOCKER_REGISTRY_USER}" "${DOCKER_REGISTRY_PASSWORD}" | base64 | tr -d '\n')
harbor_auth=$(printf "%s:%s" "${HARBOR_REGISTRY_USER}" "${HARBOR_REGISTRY_PASSWORD}" | base64 | tr -d '\n')

# Define the JSON structure for multiple registries
json=$(cat <<EOF
{
  "auths": {
    "${DOCKER_REGISTRY}": {
      "auth": "${docker_auth}"
    },
    "${HARBOR_REGISTRY}": {
      "auth": "${harbor_auth}"
    }
  }
}
EOF
)

# Write the JSON structure to the config file
echo "${json}" > /kaniko/.docker/config.json