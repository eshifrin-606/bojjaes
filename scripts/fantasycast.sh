#!/usr/bin/env bash
#
# Show two rosters side by side for one season and week, against a locally
# running server.
#
#   go run ./cmd/server
#   scripts/fantasycast.sh 2025 14 wood        # bojjaes vs wood
#   scripts/fantasycast.sh 2025 15 aroma bojjaes
#
# Each column is a full scripts/scores.sh report, so roster resolution, scoring,
# and the report itself live in exactly one place. Each team is scored in its
# own request, which is also why the server's per-request player cap applies per
# roster rather than per matchup.
#
# Nothing computes a margin: a starter whose game has not kicked off is
# indistinguishable from one who was inactive, so a difference printed on Sunday
# would read as a settled result. The two TOTAL lines face each other instead.
set -euo pipefail

# The gap between the left column's widest line and the right column's text.
gutter=2

usage() {
	echo "usage: $(basename "$0") <season> <week> <team> [team2]" >&2
	echo "  one team is read as the bojjaes' opponent; two are shown as given" >&2
	echo "  left column is the bojjaes, or the first team named" >&2
	echo "  env: SERVER (default http://localhost:8080)" >&2
	exit 2
}

[[ $# -ge 3 && $# -le 4 ]] || usage

season=$1
week=$2
[[ $season =~ ^[0-9]+$ && $week =~ ^[0-9]+$ ]] || usage

if (($# == 3)); then
	left_team=bojjaes
	right_team=$3
else
	left_team=$3
	right_team=$4
fi

scores=$(dirname "$0")/scores.sh

# Both reports are captured before anything prints, so a failure of either team
# exits with that team's message rather than half a matchup.
left_report=$("$scores" "$season" "$week" "$left_team")
right_report=$("$scores" "$season" "$week" "$right_team")

left_lines=()
right_lines=()
# bash 3.2 on macOS: no mapfile. IFS= and -r keep blank separator lines and
# padded fields intact.
while IFS= read -r line; do left_lines+=("$line"); done <<<"$left_report"
while IFS= read -r line; do right_lines+=("$line"); done <<<"$right_report"

# Computed, not fixed: scores.sh pads names to a minimum width without
# truncating, so one long name would otherwise push into the right column.
left_width=0
for line in "${left_lines[@]}"; do
	((${#line} > left_width)) && left_width=${#line}
done
((left_width += gutter))

echo "season $season week $week"
echo

rows=${#left_lines[@]}
((${#right_lines[@]} > rows)) && rows=${#right_lines[@]}

for ((i = 0; i < rows; i++)); do
	left=${left_lines[$i]:-}
	right=${right_lines[$i]:-}
	if [[ -n $right ]]; then
		printf '%-*s%s\n' "$left_width" "$left" "$right"
	else
		printf '%s\n' "$left"
	fi
done
