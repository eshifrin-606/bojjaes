package web

import (
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
)

func TestGameState(t *testing.T) {
	kickoff := time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)
	cin := func(state games.State, period, status string) games.Week {
		return games.NewWeek(2026, 6, []games.Game{
			{Home: "CIN", Away: "PIT", State: state, Kickoff: kickoff, Period: period, Status: status},
		})
	}

	tests := []struct {
		name        string
		team        string
		week        games.Week
		available   bool
		wantText    string
		wantKickoff time.Time
	}{
		{name: "upcoming shows its kickoff", team: "CIN", week: cin(games.Upcoming, "", "pre_game"), available: true, wantKickoff: kickoff},
		{name: "in progress with a numbered quarter", team: "CIN", week: cin(games.InProgress, "3", "in_game"), available: true, wantText: "Q3"},
		{name: "in progress at halftime", team: "CIN", week: cin(games.InProgress, "HALF", "in_game"), available: true, wantText: "HALF"},
		{name: "in progress before any quarter is sent", team: "CIN", week: cin(games.InProgress, "", "in_game"), available: true, wantText: "Live"},
		{name: "complete", team: "CIN", week: cin(games.Complete, "F", "complete"), available: true, wantText: "Final"},
		{name: "complete in overtime", team: "CIN", week: cin(games.Complete, "F/OT", "complete"), available: true, wantText: "Final/OT"},
		{name: "complete before any quarter is sent", team: "CIN", week: cin(games.Complete, "", "complete"), available: true, wantText: "Final"},
		{name: "complete with unrecognized quarter text", team: "CIN", week: cin(games.Complete, "4", "complete"), available: true, wantText: "4"},
		{name: "other shows the provider's status", team: "CIN", week: cin(games.Other, "", "postponed"), available: true, wantText: "postponed"},
		{name: "no game in a week with games is a bye", team: "KC", week: cin(games.InProgress, "3", "in_game"), available: true, wantText: "Bye"},
		{name: "a free agent has no game", team: "FA", week: cin(games.InProgress, "3", "in_game"), available: true},
		{name: "an empty week shows nothing, not a bye", team: "KC", week: games.NewWeek(2026, 6, nil), available: true},
		{name: "unavailable games show nothing for a team with a game", team: "CIN", week: cin(games.Complete, "F", "complete")},
		{name: "unavailable games show nothing for a team without one", team: "KC", week: cin(games.Complete, "F", "complete")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, ko := gameState(tt.team, tt.week, tt.available)
			if text != tt.wantText || !ko.Equal(tt.wantKickoff) {
				t.Errorf("gameState(%q) = (%q, %v), want (%q, %v)", tt.team, text, ko, tt.wantText, tt.wantKickoff)
			}
		})
	}
}
