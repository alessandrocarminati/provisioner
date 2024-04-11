#!/bin/bash

patterns=(
    "j784s4-evm login:"
    "root@j784s4-evm:~#"
)
actions=(
    "root"
    "ls /"
)
pos=0

while [ $pos -lt ${#patterns[@]} ]; do
    read -r input_str

    if [ -z "$input_str" ]; then
        continue
    fi

    if [[ $input_str =~ ${patterns[$pos]} ]]; then
        echo "found $pos" >&2
        echo "${actions[$pos]}"
        ((pos++))
    fi
done

echo "script terminated" >&2
sleep 2
