#!/bin/bash

# rewrites the title as Kirby front-matter (in-place)
addFrontMatter() {
    title=$(head -1 "$1" | sed -r 's/^# //')
    body=$(tail -n +2 "$1")
    {
        echo "Title: $title"
        echo
        echo "----"
        echo
        echo "Text:"
        echo "$body"
    } > "$1"
}

# rewrites all markdown links to indexes (in-place)
rewriteLinks() {
    sed -i 's/\.md//g' "$1"
}

for file in $(find docs -name '*-*.md'); do
    x=$(basename "$file")
    dir=$(dirname "$file")
    f=${x%.md}
    buildDir="build/$dir/$f"
    mkdir -p "$buildDir"
    cp "$file" "$buildDir/docs.md"
    addFrontMatter "$buildDir/docs.md"
    rewriteLinks "$buildDir/docs.md"
done

# top-level index is weird exception to the structure
buildDir="build/docs/00-index"
mkdir -p "$buildDir"
cp docs/README.md "$buildDir/docs.md"
addFrontMatter "$buildDir/docs.md"
rewriteLinks "$buildDir/docs.md"

# configuration examples in JSON5 format
cp -R docs/30-configuration/examples build/docs/30-configuration/examples
