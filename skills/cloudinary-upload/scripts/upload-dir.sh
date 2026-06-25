#!/bin/bash
# Upload all images in a local directory to a flat Cloudinary folder,
# converting images to webp on storage.
#
# Usage:
#   upload-dir.sh [-f FOLDER] [-r] <local_directory>
#
#   -f FOLDER   Cloudinary asset folder to upload into (default: uploads)
#   -r          Keep original format (skip webp conversion)
#
# Files are uploaded flat (folder structure is NOT preserved). Each file's
# public_id is its name without extension.
#
# Requires: official cloudinary CLI (`cld`) and CLOUDINARY_URL env var.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FOLDER="uploads"
CONVERT_FLAG=""

while getopts ":f:r" opt; do
    case "$opt" in
        f) FOLDER="$OPTARG" ;;
        r) CONVERT_FLAG="-r" ;;
        \?) echo "Unknown option: -$OPTARG" >&2; exit 1 ;;
        :) echo "Option -$OPTARG requires an argument" >&2; exit 1 ;;
    esac
done
shift $((OPTIND - 1))

if [ "$#" -ne 1 ] || [ ! -d "$1" ]; then
    echo "Usage: upload-dir.sh [-f FOLDER] [-r] <local_directory>" >&2
    exit 1
fi

DIR="$1"

# Collect top-level files only (flat upload).
found=0
while IFS= read -r -d '' file; do
    found=1
    if [ -n "$CONVERT_FLAG" ]; then
        bash "$SCRIPT_DIR/upload.sh" -f "$FOLDER" -r "$file"
    else
        bash "$SCRIPT_DIR/upload.sh" -f "$FOLDER" "$file"
    fi
done < <(find "$DIR" -maxdepth 1 -type f -print0)

if [ "$found" -eq 0 ]; then
    echo "No files found in $DIR" >&2
    exit 1
fi
