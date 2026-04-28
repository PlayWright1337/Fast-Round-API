package models

type EventType string
type Team string

const (
	EventGoal     EventType = "goal"
	EventWinRound EventType = "win_round"

	TeamA Team = "a"
	TeamB Team = "b"
)

type EventRequest struct {
	MatchID string    `json:"match_id" binding:"required"`
	Team    Team      `json:"team" binding:"required"`
	Type    EventType `json:"type" binding:"required"`
}

type MatchState struct {
	MatchID string `json:"match_id"`
	ScoreA  int64  `json:"score_a"`
	ScoreB  int64  `json:"score_b"`
	RoundsA int64  `json:"rounds_a"`
	RoundsB int64  `json:"rounds_b"`
}
