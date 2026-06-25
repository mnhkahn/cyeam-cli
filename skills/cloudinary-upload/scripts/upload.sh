#!/bin/bash
# Upload local files to Cloudinary using the official `cld` CLI.
# - Converts images to webp on storage (format=webp)
# - Uploads into a flat target folder (default: uploads)
# - Overwrites existing assets with the same public_id
#
# Usage:
#   upload.sh [-f FOLDER] [-r] <path1> [path2] ...
#
#   -f FOLDER   Cloudinary asset folder to upload into (default: uploads)
#   -r          Keep original format (skip webp conversion)
#
# Requires: official cloudinary CLI (`cld`) and CLOUDINARY_URL env var.
set -euo pipefail

FOLDER="uploads"
CONVERT_WEBP=1

while getopts ":f:r" opt; do
    case "$opt" in
        f) FOLDER="$OPTARG" ;;
        r) CONVERT_WEBP=0 ;;
        \?) echo "Unknown option: -$OPTARG" >&2; exit 1 ;;
        :) echo "Option -$OPTARG requires an argument" >&2; exit 1 ;;
    esac
done
shift $((OPTIND - 1))

if ! command -v cld >/dev/null 2>&1; then
    echo "cld (Cloudinary CLI) not found, installing via pip3..." >&2
    pip3 install --user cloudinary-cli >&2
    # pip --user installs to ~/.local/bin; ensure it's in PATH
    export PATH="$HOME/.local/bin:$PATH"
    if ! command -v cld >/dev/null 2>&1; then
        echo "Error: pip install succeeded but cld still not in PATH. Add ~/.local/bin to your PATH, or: pip3 install cloudinary-cli" >&2
        exit 1
    fi
fi

if [ -z "${CLOUDINARY_URL:-}" ]; then
    echo "Error: CLOUDINARY_URL is not set." >&2
    echo "Set it like: export CLOUDINARY_URL=cloudinary://<api_key>:<api_secret>@<cloud_name>" >&2
    exit 1
fi

if [ "$#" -eq 0 ]; then
    echo "Usage: upload.sh [-f FOLDER] [-r] <path1> [path2] ..." >&2
    exit 1
fi

# Image extensions that get converted to webp.
is_image() {
    local lower
    lower="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
    case "$lower" in
        *.jpg|*.jpeg|*.png|*.gif|*.bmp|*.tiff|*.tif|*.webp|*.heic|*.heif|*.avif) return 0 ;;
        *) return 1 ;;
    esac
}

for path in "$@"; do
    if [ ! -f "$path" ]; then
        echo "Skip (not a file): $path" >&2
        continue
    fi

    base="$(basename "$path")"
    public_id="${base%.*}"

    params=(public_id="$public_id" folder="$FOLDER" overwrite=True unique_filename=False use_filename=True)

    if [ "$CONVERT_WEBP" -eq 1 ] && is_image "$path"; then
        params+=(format=webp)
    fi

    echo "Uploading $path -> $FOLDER/$public_id ..."
    cld uploader upload "$path" "${params[@]}"
done
