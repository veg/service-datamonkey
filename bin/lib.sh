#!/bin/bash

function manageDeps {
  # go, npx and openapi stuffs
  echo "im a stub for dependency management"
}

function getApiSpec {
  echo "Pulling down latest OpenAPI Specification"
  wget https://raw.githubusercontent.com/d-callan/api-datamonkey/refs/heads/master/dist/openapi.yaml -O openapi.yaml 
  echo "OpenAPI Specification retrieved!"
}

function generateServer {
  echo "Starting server code generation"
  npx @openapitools/openapi-generator-cli generate -i openapi.yaml -g go-gin-server -o . --skip-validate-spec
  echo "Code generation complete"
}

if [ "$#" -ge 2 ]; then
  echo "USAGE: lib.sh [command]"
elif [ "$#" -eq 1 ]; then
  $1
fi
