#!/bin/bash

tags=$(git tag --list)
list=$(curl -s https://proxy.golang.org/github.com/byterygon/go-release/@v/list)

for tag in $(echo $tags | tr " " "\n")
do
    for tagList in $(echo  $list | tr " " "\n")
    do
        exist=false
        if [[ "$tag" == "$tagList" ]]; 
        then
            exist=true
            break
        fi
    done
    if ! $exist 
    then
        sleep 5s
        curl -s --silent "https://proxy.golang.org/github.com/byterygon/go-release/@v/$tag.info"
    fi
done