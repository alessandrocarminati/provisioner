#!/bin/bash

patterns=("hello" "world" "example")

while IFS= read -r line; do
    echo "process line _$line"
    for pattern in "${patterns[@]}"; do
        if [[ $line =~ $pattern ]]; then
            case $pattern in
                "hello")
                    echo "Matched 'hello'!"
                    ;;
                "world")
                    echo "Matched 'world'!"
                    ;;
                "example")
                    echo "Matched 'example'!"
                    ;;
            esac
        fi
    done
done
