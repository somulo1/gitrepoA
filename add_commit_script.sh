#!/bin/bash

# Script to add and commit each untracked file one by one

git ls-files --others --exclude-standard | while read -r file; do
    git add "$file"
    git commit -m "Add $file"
done