#!/bin/bash

set -e

help() {
    cat <<- EOF
Usage: TAG=tag $0
Creates git tags for public Go packages.
VARIABLES:
  TAG        git tag, for example, v1.0.0
EOF
    exit 0
}

if [ -z "$TAG" ]
then
    printf "TAG env var is required\n\n";
    help
fi

git tag ${TAG}
git push origin ${TAG}
