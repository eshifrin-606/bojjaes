#!/usr/bin/env bash
#
# Score a roster for one season and week against a locally running server.
#
#   go run ./cmd/server
#   scripts/scores.sh 2025 14 [2025-wk14/bojjaes.csv]
#
# The players file is "id,name" per line; the name is local labeling only — the
# server never sees it, so a wrong name pairs silently with the wrong stats. Its
# first nine records are printed as the starting lineup and totalled; the rest
# are printed as bench and are not.
set -euo pipefail

# Starters are the first records in the file, in file order. Nothing validates
# that they form a legal lineup — the file carries no positions, so the roster
# file is the lineup card and reordering two lines changes the total.
lineup_size=9

usage() {
	echo "usage: $(basename "$0") <season> <week> [players-file]" >&2
	echo "  env: SERVER (default http://localhost:8080)" >&2
	exit 2
}

[[ $# -ge 2 && $# -le 3 ]] || usage

season=$1
week=$2
players_file=${3:-"$(dirname "$0")/2025-wk14/bojjaes.csv"}
server=${SERVER:-http://localhost:8080}

[[ $season =~ ^[0-9]+$ && $week =~ ^[0-9]+$ ]] || usage
[[ -f $players_file ]] || { echo "no such players file: $players_file" >&2; exit 1; }

ids=()
names=()
# A players file whose final line has no newline still yields that record.
while IFS=, read -r id name || [[ -n $id ]]; do
	id=${id// /}
	[[ -z $id || $id == \#* ]] && continue
	ids+=("$id")
	names+=("${name# }")
done <"$players_file"

((${#ids[@]})) || { echo "no players in $players_file" >&2; exit 1; }

request=$(jq -n --argjson season "$season" --argjson week "$week" \
	'$ARGS.positional as $ids | {season: $season, week: $week, player_ids: $ids}' \
	--args "${ids[@]}")

response=$(curl -sS --fail-with-body -X POST "$server/scores" \
	-H 'Content-Type: application/json' -d "$request") || {
	echo "request to $server/scores failed: $response" >&2
	exit 1
}

# A player the server has no stats for is not a zero — see README.
points_for() {
	jq -r --arg id "$1" \
		'first(.scores[] | select(.stats.player_id == $id) | .points) // "no stats"' \
		<<<"$response"
}

print_players() {
	local i
	for ((i = $1; i < $2 && i < ${#ids[@]}; i++)); do
		printf '%-24s %s\n' "${names[$i]}" "$(points_for "${ids[$i]}")"
	done
}

echo "$(jq -r '"season \(.season) week \(.week)"' <<<"$response")"

echo
echo "STARTERS"
print_players 0 "$lineup_size"

# Starters the server had no stats for contribute nothing. The total carries no
# marker for them: their "no stats" lines sit directly above it.
total=$(jq -r '$ARGS.positional as $ids
	| [.scores[] | select(.stats.player_id | IN($ids[])) | .points]
	| add // 0' \
	--args "${ids[@]:0:$lineup_size}" <<<"$response")
printf '%-24s %s\n' "TOTAL" "$total"

echo
echo "BENCH"
print_players "$lineup_size" "${#ids[@]}"
