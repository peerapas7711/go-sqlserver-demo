package timeattendance

type Item struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Payload struct {
	TimeAttendance []Item `json:"time_attendance"`
	ScoreMax       int    `json:"score_max"`
	ScoreObtained  int    `json:"score_obtained"`
	PenaltyTotal   int    `json:"penalty_total"`
}
