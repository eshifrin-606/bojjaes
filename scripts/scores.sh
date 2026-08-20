#!/usr/bin/env bash
#
# Score a roster for one season and week against a locally running server.
#
#   go run ./cmd/server
#   scripts/scores.sh 2025 14 [players.csv]
#
# The players file is "id,name" per line; the name is local labeling only — the
# server never sees it, so a wrong name pairs silently with the wrong stats.
set -euo pipefail

usage() {
	echo "usage: $(basename "$0") <season> <week> [players-file]" >&2
	echo "  env: SERVER (default http://localhost:8080)" >&2
	exit 2
}

[[ $# -ge 2 && $# -le 3 ]] || usage

season=$1
week=$2
players_file=${3:-"$(dirname "$0")/players.csv"}
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

echo "$(jq -r '"season \(.season) week \(.week)"' <<<"$response")"

for i in "${!ids[@]}"; do
	points=$(jq -r --arg id "${ids[$i]}" \
		'first(.scores[] | select(.stats.player_id == $id) | .points) // "no stats"' \
		<<<"$response")
	printf '%-24s %s\n' "${names[$i]}" "$points"
done
