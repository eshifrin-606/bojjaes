package web

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/nflweek"
)

// CurrentWeekSource supplies the provider's current NFL week. Declared here,
// by the consumer, for the reason StatsSource is.
type CurrentWeekSource interface {
	AsOf(context.Context) (nflweek.Current, time.Time, error)
}

//go:embed lineup.html
var lineupFS embed.FS

var lineupPage = template.Must(template.ParseFS(lineupFS, "lineup.html"))

type lineupView struct {
	Season, Week int
	NoWeek       bool
	Team         string
	Teams        []string
	Slots        []slotSelect

	StartingUnreadable bool
	Problems           []string
	CSVLink            string
}

// slotSelect is one slot's select. Its name is the slot's query parameter, so
// the form submits the URL a proposal is shared as.
type slotSelect struct {
	Label, Name string
	Options     []playerOption
	Empty       bool
	Card        playerCard
}

// playerCard is the chosen player as the matchup page shows a starter. It is
// zero for a slot on the empty option.
type playerCard struct {
	Name, Position, Team, Game string
}

type playerOption struct {
	ID, Text string
	Selected bool
}

// slotSelects offers each slot the pool's players it accepts, in pool order,
// with the chosen player selected. A chosen player the slot does not accept
// is offered there too, so a shared URL shows what it proposed.
func slotSelects(pool []lineup.Player, chosen [lineup.StarterCount]lineup.Player, weekGames games.Week, gamesAvailable bool) []slotSelect {
	var selects []slotSelect
	for i, slot := range lineup.Slots() {
		s := slotSelect{Label: string(slot), Name: strings.ToLower(string(slot))}
		offer := func(p lineup.Player) {
			card := playerCard{Name: p.Name, Position: p.Position, Team: p.Team, Game: playerGameText(p, weekGames, gamesAvailable)}
			selected := p.ID == chosen[i].ID
			s.Options = append(s.Options, playerOption{ID: p.ID, Text: card.optionText(), Selected: selected})
			if selected {
				s.Card = card
			}
		}
		for _, p := range pool {
			if slot.Accepts(p.Position) {
				offer(p)
			}
		}
		if chosen[i].ID != "" && !slot.Accepts(chosen[i].Position) {
			offer(chosen[i])
		}
		s.Empty = !slices.ContainsFunc(s.Options, func(o playerOption) bool { return o.Selected })
		selects = append(selects, s)
	}
	return selects
}

// playerGameText states a player's game as scoreColumn's meta line does, but
// as plain text: an <option> cannot hold the <time> the matchup page
// localises, and the card shows what the option says.
func playerGameText(p lineup.Player, weekGames games.Week, gamesAvailable bool) string {
	game, kickoff := gameState(p.Team, weekGames, gamesAvailable)
	if !kickoff.IsZero() {
		game = kickoffText(kickoff.In(chicagoLoc))
	}
	return game
}

func (c playerCard) optionText() string {
	text := c.Name + " · " + c.Position + " · " + c.Team
	if c.Game != "" {
		text += " · " + c.Game
	}
	return text
}

// startingLineup is the team's lineup at or before the week, in slot order,
// or nil when it has none.
func startingLineup(tree *lineup.Tree, team string, season, week int) ([]lineup.Player, error) {
	s, w, err := tree.LatestLineup(team, season, week)
	if errors.Is(err, lineup.ErrNoLineup) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l, err := tree.Read(s, w, team)
	if err != nil {
		return nil, err
	}
	return l.Starters(), nil
}

// optionPool is the roster, then any starter not on it: a starter the roster
// file has dropped must still show in the slot he holds.
func optionPool(roster, starters []lineup.Player) []lineup.Player {
	pool := slices.Clone(roster)
	for _, s := range starters {
		if !slices.ContainsFunc(roster, func(p lineup.Player) bool { return p.ID == s.ID }) {
			pool = append(pool, s)
		}
	}
	return pool
}

// proposedIDs reads one ID per slot from the slot-named parameters, filling
// repeated slots in URL order, and names any slot given more values than it
// has places. ok is false when no slot parameter is present: the page then
// shows the starting lineup.
func proposedIDs(q url.Values) (ids [lineup.StarterCount]string, problems []string, ok bool) {
	slots := lineup.Slots()
	places := map[lineup.Slot]int{}
	for i, slot := range slots {
		values, present := q[strings.ToLower(string(slot))]
		ok = ok || present
		if n := places[slot]; n < len(values) {
			ids[i] = values[n]
		}
		places[slot]++
	}
	// Repeated slots sit next to each other, so Compact leaves each kind once.
	for _, slot := range slices.Compact(slots[:]) {
		if n := len(q[strings.ToLower(string(slot))]); n > places[slot] {
			problems = append(problems, fmt.Sprintf("too many %s players: %d for %d slots", slot, n, places[slot]))
		}
	}
	return ids, problems, ok
}

// checkProposal looks up each proposed ID in the pool, listing what keeps the
// slots from being a lineup. An unknown ID leaves its slot empty.
func checkProposal(ids [lineup.StarterCount]string, pool []lineup.Player) (players [lineup.StarterCount]lineup.Player, problems []string) {
	for i, slot := range lineup.Slots() {
		if ids[i] == "" {
			problems = append(problems, fmt.Sprintf("%s slot is empty", slot))
			continue
		}
		at := slices.IndexFunc(pool, func(p lineup.Player) bool { return p.ID == ids[i] })
		if at < 0 {
			problems = append(problems, fmt.Sprintf("%s: no player %s on the roster or starting lineup", slot, ids[i]))
			continue
		}
		p := pool[at]
		if !slot.Accepts(p.Position) {
			problems = append(problems, fmt.Sprintf("%s (%s) cannot play %s", p.Name, p.Position, slot))
		}
		if slices.ContainsFunc(players[:i], func(q lineup.Player) bool { return q.ID == p.ID }) {
			problems = append(problems, p.Name+" is in two slots")
		}
		players[i] = p
	}
	return players, problems
}

// csvLink is the download's URL. Its query is built in slot order, not by
// url.Values.Encode, which sorts keys, so it reads like the form's own URL.
func csvLink(team string, l lineup.Lineup) string {
	link := "/lineup.csv?team=" + url.QueryEscape(team)
	slots := lineup.Slots()
	for i, p := range l.Starters() {
		link += "&" + strings.ToLower(string(slots[i])) + "=" + url.QueryEscape(p.ID)
	}
	return link
}

// lineupForm is what both lineup routes resolve from a request before they
// part: the page renders it, the download writes its lineup.
type lineupForm struct {
	team               string
	season, week       int
	noWeek             bool
	pool               []lineup.Player
	chosen             [lineup.StarterCount]lineup.Player
	proposed           bool
	problems           []string
	startingUnreadable bool
	lineup             lineup.Lineup
	complete           bool
}

// resolveLineupForm runs the team, current-week, pool, and proposal steps
// both routes share. When ok is false it has already answered the request.
func resolveLineupForm(w http.ResponseWriter, r *http.Request, rosters *lineup.Rosters, tree *lineup.Tree, current CurrentWeekSource) (form lineupForm, ok bool) {
	// Refused before any fetch: a typo or a guess must not reach Sleeper.
	form.team = r.URL.Query().Get("team")
	if form.team == "" {
		form.team = lineup.OurTeam
	}
	if !slices.Contains(rosters.Teams(), form.team) {
		http.Error(w, "no such team", http.StatusBadRequest)
		return form, false
	}

	cur, _, err := current.AsOf(r.Context())
	if err != nil {
		log.Printf("reading the current week: %v", err)
		// Never a form for a guessed week: it would file a lineup for the
		// wrong one.
		http.Error(w, "the current week could not be read", http.StatusBadGateway)
		return form, false
	}
	var settable bool
	form.season, form.week, settable = cur.Settable()
	form.noWeek = !settable || form.week > lineup.LastWeek
	if form.noWeek {
		return form, true
	}

	roster, err := rosters.Read(form.team)
	if err != nil {
		// Our embedded roster is wrong, not the request: a team the picker
		// lists must read.
		log.Printf("reading %s roster: %v", form.team, err)
		http.Error(w, "that team's roster could not be read", http.StatusInternalServerError)
		return form, false
	}

	starters, err := startingLineup(tree, form.team, form.season, form.week)
	if err != nil {
		// The slots can still be set by hand, so a bad file is a notice, not
		// a failed page.
		log.Printf("reading %s starting lineup for %d week %d: %v", form.team, form.season, form.week, err)
		form.startingUnreadable = true
	}
	copy(form.chosen[:], starters)
	form.pool = optionPool(roster, starters)

	proposed, overfull, isProposal := proposedIDs(r.URL.Query())
	form.proposed = isProposal
	if isProposal {
		form.chosen, form.problems = checkProposal(proposed, form.pool)
		form.problems = append(overfull, form.problems...)
	}
	if len(form.problems) == 0 {
		form.lineup, err = lineup.FromSlots(form.chosen)
		form.complete = err == nil
		// A starting lineup with an empty slot is no problem to report: it
		// just has no summary until the slots are filled.
		if err != nil && isProposal {
			form.problems = append(form.problems, err.Error())
		}
	}
	return form, true
}

// LineupHandler renders the lineup form for the current week.
func LineupHandler(rosters *lineup.Rosters, tree *lineup.Tree, current CurrentWeekSource, gameSource GamesSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form, ok := resolveLineupForm(w, r, rosters, tree, current)
		if !ok {
			return
		}
		view := lineupView{
			Season:             form.season,
			Week:               form.week,
			NoWeek:             form.noWeek,
			Team:               form.team,
			Teams:              rosters.Teams(),
			StartingUnreadable: form.startingUnreadable,
			Problems:           form.problems,
		}
		if !form.noWeek {
			// Read last, and a failure only drops the game text: the form
			// sets a lineup whether or not Sleeper says when anyone plays.
			weekGames, _, err := gameSource.AsOf(r.Context(), form.season, form.week)
			gamesAvailable := err == nil
			if err != nil {
				log.Printf("fetching games for %d week %d: %v", form.season, form.week, err)
			}
			view.Slots = slotSelects(form.pool, form.chosen, weekGames, gamesAvailable)
		}
		if form.complete {
			view.CSVLink = csvLink(form.team, form.lineup)
		}

		var buf bytes.Buffer
		if err := lineupPage.Execute(&buf, view); err != nil {
			log.Printf("rendering lineup form: %v", err)
			http.Error(w, "the page could not be rendered", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	})
}

// resolveProposedLineup is resolveLineupForm for the routes that answer the
// lineup itself, which need a complete, valid proposal for a week to set.
func resolveProposedLineup(w http.ResponseWriter, r *http.Request, rosters *lineup.Rosters, tree *lineup.Tree, current CurrentWeekSource) (lineupForm, bool) {
	form, ok := resolveLineupForm(w, r, rosters, tree, current)
	if !ok {
		return form, false
	}
	if form.noWeek {
		http.Error(w, "no week to set", http.StatusNotFound)
		return form, false
	}
	if !form.proposed || !form.complete {
		http.Error(w, "not a complete, valid lineup: "+strings.Join(form.problems, "; "), http.StatusBadRequest)
		return form, false
	}
	return form, true
}

// LineupTextHandler answers a proposed lineup as text to paste into the
// league's thread.
func LineupTextHandler(rosters *lineup.Rosters, tree *lineup.Tree, current CurrentWeekSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form, ok := resolveProposedLineup(w, r, rosters, tree, current)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s · %d week %d\n", form.team, form.season, form.week)
		slots := lineup.Slots()
		for i, p := range form.lineup.Starters() {
			fmt.Fprintf(w, "%s %s (%s)\n", slots[i], p.Name, p.Team)
		}
	})
}

// LineupCSVHandler answers a proposed lineup as a lineup file.
func LineupCSVHandler(rosters *lineup.Rosters, tree *lineup.Tree, current CurrentWeekSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form, ok := resolveProposedLineup(w, r, rosters, tree, current)
		if !ok {
			return
		}
		var buf bytes.Buffer
		if err := form.lineup.WriteCSV(&buf); err != nil {
			log.Printf("writing %s lineup file: %v", form.team, err)
			http.Error(w, "the lineup file could not be written", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, form.team))
		w.Write(buf.Bytes())
	})
}
