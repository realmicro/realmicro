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

sed -i "" "s/\(\"version\": \)\"[^\"]*\"/\1\"${TAG#v}\"/" package.json

git add -u
git commit -m "chore: tag $TAG (tag.sh)"
git push origin master

git tag ${TAG}
git push origin ${TAG}
