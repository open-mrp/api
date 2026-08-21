#!/bin/bash

# check-no-binaries.sh
# Fails when a tracked file is a compiled executable or is large enough to bloat every clone.
# tools/apidocs/apidocs reached main at 51 MB and stayed in history across five commits; .gitignore
# only stops the paths someone thought to list, so this is the check that catches the rest.

set -uo pipefail

RED='\033[0;31m'
NC='\033[0m'

MAX_BYTES=${MAX_TRACKED_FILE_BYTES:-2097152} # 2 MiB

failed=0

while IFS= read -r path; do
    [ -f "$path" ] || continue

    kind=$(file -b --mime-type "$path")
    case "$kind" in
    application/x-mach-binary | application/x-executable | application/x-sharedlib | application/x-dosexec | application/x-object)
        echo -e "${RED}[ERROR]${NC} $path is a compiled binary ($kind). Build artifacts must not be committed."
        failed=1
        continue
        ;;
    esac

    size=$(wc -c <"$path" | tr -d ' ')
    if [ "$size" -gt "$MAX_BYTES" ]; then
        echo -e "${RED}[ERROR]${NC} $path is $((size / 1024)) KiB, over the $((MAX_BYTES / 1024)) KiB limit for a tracked file."
        echo "        Generate it at build time, or raise MAX_TRACKED_FILE_BYTES if it genuinely belongs in the repo."
        failed=1
    fi
done < <(git ls-files)

if [ "$failed" -ne 0 ]; then
    echo ""
    echo "Remove the file and untrack it:  git rm --cached <path> && echo '<path>' >> .gitignore"
    exit 1
fi

echo "No committed binaries or oversized files."
