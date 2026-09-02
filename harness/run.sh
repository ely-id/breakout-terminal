#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "usage: $0 /path/to/original-breakout-plugin" >&2
    exit 2
fi

SOURCE="$(realpath "$1")"
ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

cd "$ROOT"

for command in herdr codex jq; do
    command -v "$command" >/dev/null 2>&1 || {
        echo "missing dependency: $command" >&2
        exit 1
    }
done

if [[ ! -d "$SOURCE" ]]; then
    echo "source plugin does not exist: $SOURCE" >&2
    exit 1
fi

mkdir -p reference docs

rm -rf reference/plugin-source
mkdir -p reference/plugin-source
cp -a "$SOURCE"/. reference/plugin-source/

echo "source snapshot: $SOURCE"

created="$(
    herdr workspace create \
        --cwd "$ROOT" \
        --label breakout-port \
        --no-focus
)"

mechanics_pane="$(
    printf '%s\n' "$created" |
        jq -r '.result.root_pane.pane_id'
)"

split="$(
    herdr pane split "$mechanics_pane" \
        --direction right \
        --no-focus
)"

portability_pane="$(
    printf '%s\n' "$split" |
        jq -r '.result.pane.pane_id'
)"

split="$(
    herdr pane split "$mechanics_pane" \
        --direction down \
        --no-focus
)"

builder_pane="$(
    printf '%s\n' "$split" |
        jq -r '.result.pane.pane_id'
)"

herdr agent start mechanics \
    --kind codex \
    --pane "$mechanics_pane"

herdr agent start portability \
    --kind codex \
    --pane "$portability_pane"

herdr agent start builder \
    --kind codex \
    --pane "$builder_pane"

echo
echo "=== Phase 1: parallel reverse engineering ==="

prompt_and_wait() {
    local agent="$1"
    local prompt_file="$2"
    local result
    local status

    result="$(
        herdr agent prompt "$agent" \
            "$(cat "$prompt_file")" \
            --wait \
            --until idle \
            --until done \
            --until blocked
    )"

    printf '%s\n' "$result"

    status="$(
        printf '%s\n' "$result" |
            jq -r '.result.agent.agent_status // empty'
    )"

    case "$status" in
        idle|done)
            ;;
        blocked)
            echo "agent $agent is blocked and requires interaction" >&2
            return 1
            ;;
        *)
            echo "agent $agent ended in unexpected status: ${status:-unknown}" >&2
            return 1
            ;;
    esac
}

prompt_and_wait mechanics harness/prompts/mechanics.md &
mechanics_job=$!

prompt_and_wait portability harness/prompts/portability.md &
portability_job=$!

phase1_failed=0

wait "$mechanics_job" || phase1_failed=1
wait "$portability_job" || phase1_failed=1

if (( phase1_failed != 0 )); then
    echo "Phase 1 failed." >&2
    exit 1
fi

test -s docs/GAME_SPEC.md
test -s docs/PORT_SPEC.md

echo
echo "=== Phase 2: implementation ==="

herdr agent prompt builder \
    "$(cat harness/prompts/builder.md)" \
    --wait \
    --until idle \
    --until done

test -s docs/BUILD_REPORT.md

echo
echo "=== Phase 3: independent review ==="

herdr agent prompt portability \
    "$(cat harness/prompts/reviewer.md)" \
    --wait \
    --until idle \
    --until done

test -s docs/REVIEW.md

echo
echo "=== Phase 4: corrections ==="

herdr agent prompt builder \
    "$(cat harness/prompts/fix.md)" \
    --wait \
    --until idle \
    --until done

echo
echo "=== Phase 5: final gate ==="

herdr agent prompt portability \
    "$(cat harness/prompts/final-review.md)" \
    --wait \
    --until idle \
    --until done

test -s docs/FINAL_REVIEW.md

echo
echo "======================================"
head -n 1 docs/FINAL_REVIEW.md
echo "======================================"

if grep -qx 'VERDICT: PASS' docs/FINAL_REVIEW.md; then
    echo "Breakout port passed the harness."
else
    echo "Breakout port requires another iteration."
    exit 1
fi
