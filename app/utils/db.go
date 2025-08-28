package utils

import (
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB
var DBPath string

func Connect(path string, readOnly bool) error {
	mode := ""
	if readOnly {
		mode = "?mode=ro"
	}
	db, err := sql.Open("sqlite3", path+mode)
	if err != nil {
		return err
	}
	DB = db
	DBPath = path
	return nil
}

func QuoteIdentifier(name string) string {
	// 仅允许字母、数字、下划线，且不能以数字开头
	if !isValidIdentifier(name) {
		panic("invalid table name")
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func isValidIdentifier(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9' && i > 0:
		case r == '_':
		default:
			return false
		}
	}
	return true
}
