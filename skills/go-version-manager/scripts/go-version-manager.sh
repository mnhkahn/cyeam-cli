#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  go-version-manager.sh requirement [project-dir]
  go-version-manager.sh manager
  go-version-manager.sh list
  go-version-manager.sh select [project-dir]
  go-version-manager.sh shell-command [project-dir]
  go-version-manager.sh switch-local [project-dir]
  go-version-manager.sh exec [project-dir] -- <go-args...>
EOF
}

project_go_mod() {
  local project_dir="${1:-.}"
  local mod_file="$project_dir/go.mod"
  if [[ ! -f "$mod_file" ]]; then
    echo "go.mod not found: $mod_file" >&2
    return 1
  fi
  printf '%s\n' "$mod_file"
}

read_requirement() {
  local mod_file
  mod_file="$(project_go_mod "${1:-.}")"
  local go_version toolchain_version
  go_version="$(awk '$1 == "go" { print $2; exit }' "$mod_file")"
  toolchain_version="$(awk '$1 == "toolchain" { sub(/^go/, "", $2); print $2; exit }' "$mod_file")"
  if [[ -z "$go_version" ]]; then
    echo "missing go directive in $mod_file" >&2
    return 1
  fi
  printf 'go=%s\n' "$go_version"
  printf 'toolchain=%s\n' "$toolchain_version"
}

version_of() {
  local binary="$1"
  "$binary" version 2>/dev/null | awk '{ sub(/^go/, "", $3); print $3 }'
}

discover() {
  local -a candidates=()
  if command -v goenv >/dev/null 2>&1; then
    local goenv_version goenv_prefix
    while IFS= read -r goenv_version; do
      [[ -n "$goenv_version" ]] || continue
      goenv_prefix="$(goenv prefix "$goenv_version" 2>/dev/null || true)"
      [[ -x "$goenv_prefix/bin/go" ]] && candidates+=("$goenv_prefix/bin/go")
    done < <(goenv versions --bare 2>/dev/null || true)
  fi

  local path_go=""
  path_go="$(command -v go 2>/dev/null || true)"
  [[ -n "$path_go" ]] && candidates+=("$path_go")

  shopt -s nullglob
  candidates+=(
    "$HOME"/sdk/go*/bin/go
    "$HOME"/.local/go*/bin/go
    /usr/local/go*/bin/go
    /opt/homebrew/opt/go*/bin/go
  )
  shopt -u nullglob

  local seen="|"
  local candidate resolved version
  for candidate in "${candidates[@]}"; do
    [[ -x "$candidate" ]] || continue
    resolved="$candidate"
    if command -v realpath >/dev/null 2>&1; then
      resolved="$(realpath "$candidate" 2>/dev/null || printf '%s' "$candidate")"
    fi
    [[ "$seen" == *"|$resolved|"* ]] && continue
    version="$(version_of "$candidate")"
    [[ -n "$version" ]] || continue
    printf '%s\t%s\n' "$version" "$candidate"
    seen+="$resolved|"
  done
}

manager_name() {
  if command -v goenv >/dev/null 2>&1; then
    printf 'goenv\n'
  else
    printf 'explicit-path\n'
  fi
}

version_key() {
  local version="$1"
  local major minor patch
  IFS=. read -r major minor patch <<<"${version%%-*}"
  printf '%06d%06d%06d\n' "${major:-0}" "${minor:-0}" "${patch:-0}"
}

select_binary() {
  local project_dir="${1:-.}"
  local requirement go_version toolchain_version preferred major minor
  requirement="$(read_requirement "$project_dir")"
  go_version="$(awk -F= '$1 == "go" { print $2 }' <<<"$requirement")"
  toolchain_version="$(awk -F= '$1 == "toolchain" { print $2 }' <<<"$requirement")"
  preferred="${toolchain_version:-$go_version}"
  IFS=. read -r major minor _ <<<"$preferred"

  local best_binary="" best_key="" exact_binary=""
  local version binary key
  while IFS=$'\t' read -r version binary; do
    [[ -n "$version" ]] || continue
    if [[ -n "$toolchain_version" && "$version" == "$toolchain_version" ]]; then
      exact_binary="$binary"
      break
    fi
    if [[ "$version" == "$major.$minor" || "$version" == "$major.$minor."* ]]; then
      key="$(version_key "$version")"
      if [[ -z "$best_key" || "$key" > "$best_key" ]]; then
        best_key="$key"
        best_binary="$binary"
      fi
    fi
  done < <(discover)

  if [[ -n "$exact_binary" ]]; then
    printf '%s\n' "$exact_binary"
    return 0
  fi
  if [[ -n "$best_binary" ]]; then
    printf '%s\n' "$best_binary"
    return 0
  fi
  echo "no installed Go toolchain matches $major.$minor.x" >&2
  return 2
}

selected_version() {
  local go_binary
  go_binary="$(select_binary "${1:-.}")"
  version_of "$go_binary"
}

is_goenv_binary() {
  local go_binary="$1"
  command -v goenv >/dev/null 2>&1 || return 1
  local goenv_root
  goenv_root="$(goenv root 2>/dev/null || true)"
  [[ -n "$goenv_root" && "$go_binary" == "$goenv_root/versions/"*"/bin/go" ]]
}

main() {
  local command_name="${1:-}"
  case "$command_name" in
    requirement)
      shift
      read_requirement "${1:-.}"
      ;;
    manager)
      manager_name
      ;;
    list)
      discover
      ;;
    select)
      shift
      select_binary "${1:-.}"
      ;;
    shell-command)
      shift
      local project_dir="${1:-.}"
      local go_binary go_version
      go_binary="$(select_binary "$project_dir")"
      go_version="$(version_of "$go_binary")"
      if is_goenv_binary "$go_binary"; then
        printf 'goenv shell %q\n' "$go_version"
      else
        printf 'unset GOROOT; export PATH=%q:"$PATH"\n' "$(dirname "$go_binary")"
      fi
      ;;
    switch-local)
      shift
      local project_dir="${1:-.}"
      if ! command -v goenv >/dev/null 2>&1; then
        echo "switch-local requires goenv" >&2
        return 2
      fi
      local go_binary go_version
      go_binary="$(select_binary "$project_dir")"
      go_version="$(version_of "$go_binary")"
      if ! is_goenv_binary "$go_binary"; then
        echo "selected Go $go_version is not installed in goenv" >&2
        return 2
      fi
      (cd "$project_dir" && goenv local "$go_version")
      printf 'project Go version set to %s in %s/.go-version\n' "$go_version" "${project_dir%/}"
      ;;
    exec)
      shift
      local project_dir="${1:-.}"
      shift || true
      if [[ "${1:-}" != "--" ]]; then
        echo "exec requires -- before Go arguments" >&2
        usage >&2
        return 1
      fi
      shift
      local go_binary
      go_binary="$(select_binary "$project_dir")"
      echo "using $go_binary ($(version_of "$go_binary"))" >&2
      if is_goenv_binary "$go_binary"; then
        exec env GOENV_VERSION="$(version_of "$go_binary")" go "$@"
      fi
      exec "$go_binary" "$@"
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      usage >&2
      return 1
      ;;
  esac
}

main "$@"
