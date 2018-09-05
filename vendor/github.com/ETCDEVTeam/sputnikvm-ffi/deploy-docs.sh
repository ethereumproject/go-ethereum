#!/usr/bin/env bash

cd c
doxygen
cd ..

git add -f ./c/html
TREE_OBJ_ID=`git write-tree --prefix=c/html`
git reset -- ./c/html
COMMIT_ID=`git commit-tree -p gh-pages -m "Doc deployments" $TREE_OBJ_ID`
git update-ref refs/heads/gh-pages $COMMIT_ID
