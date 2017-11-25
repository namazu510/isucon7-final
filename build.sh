#!/usr/bin/env bash
# サーバ上で実行すると、
set -euvx

branch="${1:-}"

cd ~/webapp/go/src/app/
if [ -n "$branch" ]; then
    git fetch origin
    git checkout "origin/$branch"
fi

go build -o ../../app
