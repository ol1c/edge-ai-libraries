#!/usr/bin/env bash
# Copyright (C) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -Eeuo pipefail

NATIVE_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
CONFIG_DIR=/etc/metrics-manager
EXAMPLES_DIR=/usr/share/doc/metrics-manager/examples
OPTIONS=(RAPL POWERSTAT_EXTRA TURBOSTAT TEMPERATURE INTERRUPTS NETWORK NETWORK_VIRTUAL STORAGE)
declare -A settings destinations examples

fail() {
    printf 'configure-metrics.sh: %s\n' "$*" >&2
    exit 1
}

run() {
    if (( EUID == 0 )); then
        "$@"
    else
        sudo -- "$@"
    fi
}

exists() {
    [[ -e "$1" || -L "$1" ]]
}

read_options() {
    local file=$1 line key value number=0
    [[ -r "$file" ]] || fail "Cannot read options file: $file"
    local -A seen=()
    for key in "${OPTIONS[@]}"; do
        settings[$key]=auto
    done
    while IFS= read -r line || [[ -n "$line" ]]; do
        number=$((number + 1))
        line=${line%$'\r'}
        [[ "$line" =~ ^[[:space:]]*(#.*)?$ ]] && continue
        if [[ ! "$line" =~ ^[[:space:]]*ENABLE_([A-Z_]+)[[:space:]]*=[[:space:]]*(true|false|auto)[[:space:]]*(#.*)?$ ]]; then
            fail "$file:$number: Expected ENABLE_OPTION=true|false|auto"
        fi
        key=${BASH_REMATCH[1]}
        value=${BASH_REMATCH[2]}
        [[ -v "settings[$key]" ]] || fail "$file:$number: Unknown option ENABLE_$key"
        [[ ! -v "seen[$key]" ]] || fail "$file:$number: Duplicate option ENABLE_$key"
        seen[$key]=1
        settings[$key]=$value
    done < "$file"
}

prepare_paths() {
    local key
    for key in TEMPERATURE INTERRUPTS NETWORK STORAGE; do
        destinations[$key]="$CONFIG_DIR/telegraf.d/${key,,}.conf"
        examples[$key]="$EXAMPLES_DIR/telegraf-${key,,}.conf"
    done
    destinations[NETWORK_VIRTUAL]="$CONFIG_DIR/telegraf.d/network-virtual.conf"
    examples[NETWORK_VIRTUAL]="$EXAMPLES_DIR/telegraf-network-virtual.conf"
    destinations[POWERSTAT_EXTRA]="$CONFIG_DIR/powerstat.d/extra.conf"
    examples[POWERSTAT_EXTRA]="$EXAMPLES_DIR/telegraf-powerstat-extra.conf"
    destinations[TURBOSTAT]="$CONFIG_DIR/powerstat.d/turbostat.conf"
    examples[TURBOSTAT]="$EXAMPLES_DIR/telegraf-turbostat.conf"
}

check_turbostat() {
    local binary=${1:-/usr/bin/turbostat} output
    [[ -x "$binary" ]] ||
        fail "Install $binary for the running kernel before enabling Turbostat"
    # Some versions print help to stderr and exit with status 1.
    if ! output=$("$binary" --version 2>&1); then
        fail "Cannot run $binary --version; install Turbostat tools matching the running kernel:
$output"
    fi
}

preflight() {
    local key destination
    command -v systemctl >/dev/null || fail "systemctl is required"
    if (( EUID != 0 )); then
        command -v sudo >/dev/null || fail "Run as root or install sudo"
    fi
    for key in "${OPTIONS[@]}"; do
        [[ "$key" == RAPL || "${settings[$key]}" == auto ]] && continue
        destination=${destinations[$key]}
        case "${settings[$key]}" in
            true)
                if ! exists "$destination" && ! exists "$destination.disabled"; then
                    [[ -r "${examples[$key]}" ]] ||
                        fail "Missing example ${examples[$key]}; install the current Metrics Manager package"
                fi
                ;;
            false)
                if exists "$destination" && exists "$destination.disabled"; then
                    fail "Both $destination and $destination.disabled exist; resolve the conflict first"
                fi
                ;;
        esac
    done
    if [[ "${settings[TURBOSTAT]}" == true ]]; then
        check_turbostat
    fi
}

configure_dropin() {
    local key=$1 destination=${destinations[$1]}
    case "${settings[$key]}" in
        true)
            if exists "$destination"; then
                printf 'Keeping %s\n' "$destination"
            else
                run install -d -o root -g root -m 0755 "$(dirname -- "$destination")"
                if exists "$destination.disabled"; then
                    run mv -- "$destination.disabled" "$destination"
                else
                    run install -o root -g root -m 0644 "${examples[$key]}" "$destination"
                fi
            fi
            ;;
        false)
            if exists "$destination"; then
                run mv -- "$destination" "$destination.disabled"
            fi
            ;;
    esac
}

main() {
    if [[ "${1:-}" == --help || "${1:-}" == -h ]]; then
        printf 'Usage: %s [options.conf]\nDefault: %s/metrics-options.conf\n' "$0" "$NATIVE_DIR"
        return
    fi
    (( $# <= 1 )) || fail "Usage: $0 [options.conf]"
    read_options "${1:-$NATIVE_DIR/metrics-options.conf}"
    prepare_paths
    preflight

    local key
    for key in "${OPTIONS[@]}"; do
        printf 'ENABLE_%s=%s\n' "$key" "${settings[$key]}"
        if [[ "$key" == RAPL ]]; then
            case "${settings[$key]}" in
                true) run systemctl enable --now metrics-manager-rapl.service ;;
                false) run systemctl disable --now metrics-manager-rapl.service ;;
            esac
        else
            configure_dropin "$key"
        fi
    done

    # A plain restart would also start a loaded RAPL unit just disabled above.
    run systemctl try-restart 'metrics-manager-*.service'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    trap 'printf "configure-metrics.sh: Failed at line %s: %s\n" "$LINENO" "$BASH_COMMAND" >&2' ERR
    main "$@"
fi
