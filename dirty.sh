#!/bin/sh
state="clean"
git status --untracked-files=no --porcelain|  grep -q M && state="dirty"
echo $state
