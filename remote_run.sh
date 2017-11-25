#!/usr/bin/env bash

servers=(
    isucon@app0231.isu7f.k0y.org
    isucon@app0232.isu7f.k0y.org
    isucon@app0233.isu7f.k0y.org
    isucon@app0234.isu7f.k0y.org
)

remote_run() {
    for srv in "${servers[@]}"; do
        echo "
        set -euvx
        . ~/.bashrc
        $@
        " | ssh "$srv"
    done
}

remote_run "$@"
