#!/bin/bash

# rewrites the title as Kirby front-matter (in-place)
addFrontMatter() {
    sed -i -e '1 s/^# \(.*\)$/Title: \1\n\n----\n\nText:/; t' \
        -e '1,// s//Title: \1\n\n----\n\nText:/' \
        "$1"
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
    cp "$file" "$buildDir/index.md"
    addFrontMatter "$buildDir/index.md"
    rewriteLinks "$buildDir/index.md"
done

# top-level index is weird exception to the structure
buildDir="build/docs/00-index"
mkdir -p "$buildDir"
cp docs/README.md "$buildDir/index.md"
addFrontMatter "$buildDir/index.md"
rewriteLinks "$buildDir/index.md"

# configuration examples in JSON5 format
cp -R docs/30-configuration/examples build/docs/30-configuration/examples
