package models

// Trigger 表示一个数据库触发器
type Trigger struct {
	Name       string `json:"name"`
	Table      string `json:"table"`
	Type       string `json:"type"`   // INSERT, UPDATE, DELETE
	Timing     string `json:"timing"` // BEFORE, AFTER, INSTEAD OF
	Definition string `json:"definition"`
	SQL        string `json:"sql"`
}
