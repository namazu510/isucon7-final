#!/usr/bin/env bash
# Usage: ./deploy.sh [branch] [service] ...
# 全てのサーバに、"$1"で指定したブランチのコミットをデプロイする。
# serviceには、再起動するサービス名を指定する。
#
# Example:
#   ./deploy.sh master      - masterブランチの内容をデプロイする。
#   ./deploy.sh foo nginx   - fooブランチの内容をデプロイした後、nginxを再起動する。

servers=(
    isucon@app0231.isu7f.k0y.org
    isucon@app0232.isu7f.k0y.org
    isucon@app0233.isu7f.k0y.org
    isucon@app0234.isu7f.k0y.org
)
echo "デプロイ先: ${servers[@]}"

remote_run() {
    for srv in "${servers[@]}"; do
        echo "
        set -euvx
        . ~/.bashrc
        "$@"
        " | ssh "$srv"
    done
}

set -euvx
branch=$1
shift
services=(cco.golang.service "$@")

if [ -n "$branch" ]; then
    remote_run ./build.sh "$branch"
else
    remote_run ./build.sh
fi

remote_run sudo systemctl restart "${services[@]}"
