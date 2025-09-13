#!/bin/bash

# This script finds all .go files in the current directory and its subdirectories,
# formats them with a comment containing their relative path, and concatenates
# them into a single file named export.txt.

# Define the output file name
OUTPUT_FILE="export.txt"

# Flag parsing
EXCLUDE_TESTS=false
for arg in "$@"; do
    case $arg in
        --exclude-tests)
            EXCLUDE_TESTS=true
            shift
            ;;
    esac
done

# Step 0: Delete the old export file and create a new, empty one.
>"$OUTPUT_FILE"
echo "Initialized empty '$OUTPUT_FILE'."

# Step 1: Build the find command depending on the flag
if $EXCLUDE_TESTS; then
    echo "ℹ️ Excluding *_test.go files."
    FIND_CMD='find . -type f -name "*.go" ! -name "*_test.go"'
else
    FIND_CMD='find . -type f -name "*.go"'
fi

# Step 2: Traverse the directory tree to find files
eval "$FIND_CMD" | while read -r filepath; do
    clean_path="${filepath#./}"

    {
        echo '```go'
        echo "// $clean_path"
        echo ""
        cat "$filepath"
        echo '```'
        echo ""
        echo ""
    } >>"$OUTPUT_FILE"

    echo "Appended '$clean_path' to '$OUTPUT_FILE'."
done

echo "✅ All .go files have been successfully exported to '$OUTPUT_FILE'."
