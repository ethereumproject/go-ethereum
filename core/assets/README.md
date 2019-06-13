How to create and update default chain config binary assets from JSON config
defaults.

```bash
# Install program to include external JSON files as binary resources
go get github.com/omeid/go-resources/cmd/resources
# Create dir
mkdir config/assets
# Compile JSON to assets package (avoid recompilation with the cache using
package)
~/gocode/src/github.com/ethereumproject/go-ethereum resourceful-json-configs *% ⟠ resources -fmt -declare -var=DEFAULTS -package=assets -output=core/assets/assets.go core/config/*.json core/config/*.csv
```

When using Makefile, changes in `.json` and `.csv` files will trigger rebuilding of binary assets.
