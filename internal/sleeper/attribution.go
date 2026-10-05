package sleeper

import (
	"regexp"
	"strconv"
	"strings"
)

type fumblePair struct {
	Fumbler string
	Forcer  string
}

const playerName = `[A-Z]\.[\w'-]+(?: (?:Jr\.|Sr\.|(?:III|II|IV|V)\b))?`

var forcedByRE = regexp.MustCompile(`(` + playerName + `) FUMBLES, forced by (` + playerName + `)`)

func forcedByPairs(description string) []fumblePair {
	// Text before the last "overturned" describes a play that no longer stands.
	if i := strings.LastIndex(strings.ToLower(description), "overturned"); i >= 0 {
		description = description[i:]
	}
	var pairs []fumblePair
	for _, m := range forcedByRE.FindAllStringSubmatch(description, -1) {
		pairs = append(pairs, fumblePair{Fumbler: m[1], Forcer: m[2]})
	}
	return pairs
}

var suffixRE = regexp.MustCompile(`(?i) (?:jr\.?|sr\.?|iii|ii|iv|v)$`)

// normalizeName makes names comparable: the description writes "K.Moore II"
// where a player row may say "Moore" or "Moore II".
func normalizeName(name string) string {
	return strings.ToLower(suffixRE.ReplaceAllString(strings.TrimSpace(name), ""))
}

func abbrevName(first, last string) string {
	if first == "" {
		return normalizeName(last)
	}
	return normalizeName(first[:1] + "." + last)
}

// identity names a player as play descriptions do, with the team they played
// for that week. Play rows carry only a player ID.
type identity struct{ name, team string }

type identities map[string]identity

type play struct {
	ID        string    `json:"play_id"`
	UpdatedAt int64     `json:"updated_at"`
	Metadata  playMeta  `json:"metadata"`
	PlayStats []playRow `json:"play_stats"`
}

type playMeta struct {
	Description string `json:"description"`
	Team        string `json:"team"`
	Opponent    string `json:"opponent"`
	QuarterName string `json:"quarter_name"`
	Minutes     int    `json:"time_remaining_minutes"`
	Seconds     int    `json:"time_remaining_seconds"`
}

type playRow struct {
	PlayerID string             `json:"player_id"`
	Stats    map[string]float64 `json:"stats"`
}

// forcedFumbleTurnovers maps player ID to forced fumbles that were turnovers.
// In play-by-play, idp_ff sits on the fumbler's row; the forcer is named only
// in the description.
func forcedFumbleTurnovers(plays []play, ids identities, logf func(format string, args ...any)) map[string]int {
	credits := map[string]int{}
	for _, p := range plays {
		pairs := forcedByPairs(p.Metadata.Description)
		for _, fumbler := range p.PlayStats {
			if !fumbler.isPlayer() || fumbler.Stats["idp_ff"] <= 0 || fumbler.Stats["fum_lost"] <= 0 {
				continue
			}
			forcer, ok := resolveForcer(p, fumbler, pairs, ids)
			if !ok {
				logf("forced fumble on play %s not attributed: %s", p.ID, p.Metadata.Description)
				continue
			}
			credits[forcer.PlayerID]++
		}
	}
	return credits
}

// A row absent from ids gets the zero identity, whose empty name matches no
// parsed name, so it is never the fumbler's pair or the forcer.
func resolveForcer(p play, fumbler playRow, pairs []fumblePair, ids identities) (playRow, bool) {
	fumblerID := ids[fumbler.PlayerID]
	var forcerNames []string
	for _, pair := range pairs {
		if normalizeName(pair.Fumbler) == fumblerID.name {
			forcerNames = append(forcerNames, normalizeName(pair.Forcer))
		}
	}
	if len(forcerNames) != 1 {
		return playRow{}, false
	}
	var matches []playRow
	for _, r := range p.PlayStats {
		candidate := ids[r.PlayerID]
		if r.isPlayer() && candidate.name == forcerNames[0] && candidate.team != fumblerID.team {
			matches = append(matches, r)
		}
	}
	if len(matches) != 1 {
		return playRow{}, false
	}
	return matches[0], true
}

func (r playRow) isPlayer() bool {
	_, err := strconv.Atoi(r.PlayerID)
	return err == nil
}
