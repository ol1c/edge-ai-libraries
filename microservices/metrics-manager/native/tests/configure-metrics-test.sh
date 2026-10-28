#!/usr/bin/env bash
# Copyright (C) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -Eeuo pipefail

TEST_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=../configure-metrics.sh
source "$TEST_DIR/../configure-metrics.sh"

WORK=$(mktemp -d)
trap 'rm -rf -- "$WORK"' EXIT
CONFIG_DIR="$WORK/etc"
EXAMPLES_DIR="$WORK/examples"
LOG="$WORK/commands"
mkdir -p "$CONFIG_DIR" "$EXAMPLES_DIR" "$WORK/bin"
printf '#!/bin/sh\nexit 0\n' > "$WORK/bin/systemctl"
printf '#!/bin/sh\nif [ "$1" = -- ]; then shift; fi\nexec "$@"\n' > "$WORK/bin/sudo"
chmod +x "$WORK/bin/systemctl" "$WORK/bin/sudo"
export PATH="$WORK/bin:$PATH"

# Only filesystem operations in the temporary tree are executed.
run() {
    printf '%s\n' "$*" >> "$LOG"
    case "$1" in
        systemctl) ;;
        install)
            if [[ "$2" == -d ]]; then
                mkdir -p -- "${@: -1}"
            else
                command install -m 0644 -- "${@: -2:1}" "${@: -1}"
            fi
            ;;
        mv) command "$@" ;;
        *) fail "Unexpected test command: $*" ;;
    esac
}

assert_logged() {
    grep -Fxq -- "$*" "$LOG" || fail "Command not logged: $*"
}

expect_failure() {
    local options=$1 expected=$2
    : > "$LOG"
    if (main "$options") > "$WORK/stdout" 2> "$WORK/stderr"; then
        fail "Expected failure for $options"
    fi
    grep -Fq -- "$expected" "$WORK/stderr" || fail "Missing error: $expected"
    [[ ! -s "$LOG" ]] || fail "Invalid configuration caused side effects"
}

main "$NATIVE_DIR/metrics-options.conf" > "$WORK/stdout"
[[ $(wc -l < "$LOG") == 1 ]] || fail "auto changed configuration"
assert_logged "systemctl try-restart metrics-manager-*.service"

bash "$NATIVE_DIR/configure-metrics.sh" > "$WORK/stdout"
grep -Fxq 'ENABLE_RAPL=auto' "$WORK/stdout" || fail "Default options were not loaded"
bash "$NATIVE_DIR/configure-metrics.sh" --help > "$WORK/stdout"
grep -Fq 'Usage:' "$WORK/stdout" || fail "Help was not printed"

printf 'ENABLE_RAPL=true\n' > "$WORK/options"
: > "$LOG"
main "$WORK/options" > "$WORK/stdout"
assert_logged "systemctl enable --now metrics-manager-rapl.service"
[[ $(tail -n 1 "$LOG") == 'systemctl try-restart metrics-manager-*.service' ]] ||
    fail "Services were not reloaded last"

printf 'ENABLE_RAPL=false\n' > "$WORK/options"
: > "$LOG"
main "$WORK/options" > "$WORK/stdout"
assert_logged "systemctl disable --now metrics-manager-rapl.service"
assert_logged "systemctl try-restart metrics-manager-*.service"
[[ $(wc -l < "$LOG") == 2 ]] || fail "Unexpected service operation"

prepare_paths
for key in POWERSTAT_EXTRA TEMPERATURE INTERRUPTS NETWORK STORAGE; do
    printf 'example %s\n' "$key" > "${examples[$key]}"
    printf 'ENABLE_%s=true\n' "$key" > "$WORK/options"
    main "$WORK/options" > "$WORK/stdout"
    cmp "${examples[$key]}" "${destinations[$key]}"
    printf 'local %s\n' "$key" > "${destinations[$key]}"
    main "$WORK/options" > "$WORK/stdout"
    [[ $(< "${destinations[$key]}") == "local $key" ]] || fail "Local configuration overwritten"
    printf 'ENABLE_%s=false\n' "$key" > "$WORK/options"
    main "$WORK/options" > "$WORK/stdout"
    [[ ! -e "${destinations[$key]}" && -f "${destinations[$key]}.disabled" ]] ||
        fail "Drop-in not disabled: $key"
    main "$WORK/options" > "$WORK/stdout"
    printf 'ENABLE_%s=true\n' "$key" > "$WORK/options"
    main "$WORK/options" > "$WORK/stdout"
    [[ $(< "${destinations[$key]}") == "local $key" ]] || fail "Local configuration not restored"
    [[ ! -e "${destinations[$key]}.disabled" ]] || fail "Disabled file not restored"
done

# Test Turbostat's drop-in mapping without requiring host MSR tools.
printf 'turbostat example\n' > "${examples[TURBOSTAT]}"
settings[TURBOSTAT]=true
configure_dropin TURBOSTAT
cmp "${examples[TURBOSTAT]}" "${destinations[TURBOSTAT]}"
settings[TURBOSTAT]=false
configure_dropin TURBOSTAT
[[ -f "${destinations[TURBOSTAT]}.disabled" ]] || fail "Turbostat not disabled"

printf 'ENABLE_RAPL=true\nENABLE_UNKNOWN=false\n' > "$WORK/options"
expect_failure "$WORK/options" 'Unknown option'
printf 'ENABLE_RAPL=true\nENABLE_RAPL=false\n' > "$WORK/options"
expect_failure "$WORK/options" 'Duplicate option'
printf 'ENABLE_RAPL=TRUE\n' > "$WORK/options"
expect_failure "$WORK/options" 'Expected ENABLE_OPTION=true|false|auto'
printf 'ENABLE_RAPL=$(touch %s/injected)\n' "$WORK" > "$WORK/options"
expect_failure "$WORK/options" 'Expected ENABLE_OPTION=true|false|auto'
[[ ! -e "$WORK/injected" ]] || fail "Options executed as shell code"
expect_failure "$WORK/missing" 'Cannot read options file'

printf 'backup\n' > "${destinations[NETWORK]}.disabled"
printf 'ENABLE_RAPL=true\nENABLE_NETWORK=false\n' > "$WORK/options"
expect_failure "$WORK/options" 'resolve the conflict first'

rm -- "${examples[TURBOSTAT]}" "${destinations[TURBOSTAT]}.disabled"
printf 'ENABLE_RAPL=true\nENABLE_TURBOSTAT=true\n' > "$WORK/options"
expect_failure "$WORK/options" 'Missing example'

printf '  # comment\r\n ENABLE_RAPL = auto # unchanged\r\nENABLE_NETWORK=auto' > "$WORK/options"
: > "$LOG"
main "$WORK/options" > "$WORK/stdout"
[[ $(wc -l < "$LOG") == 1 ]] || fail "Comments, CRLF or final unterminated line misparsed"

# Service errors must fail the executable and must not reach the final reload.
printf '#!/bin/sh\nexit 42\n' > "$WORK/bin/systemctl"
printf 'ENABLE_RAPL=true\n' > "$WORK/options"
if bash "$NATIVE_DIR/configure-metrics.sh" "$WORK/options" > "$WORK/stdout" 2> "$WORK/stderr"; then
    fail "Service failure was ignored"
else
    status=$?
    [[ "$status" == 42 ]] || fail "Service exit status lost: $status"
fi
grep -Fq 'Failed at line' "$WORK/stderr" || fail "Service error was not reported"

printf 'All configure-metrics tests passed.\n'
