package gdb

import "errors"

// ErrNotSQLite - backups copy the SQLite file; a PostgreSQL database is backed up with its own tools
var ErrNotSQLite = errors.New("backups cover the built-in SQLite database; back up PostgreSQL with its own tools")

// UsingSQLite - whether the data lives in the SQLite file
func UsingSQLite() bool {
	return db != nil && db.Dialector.Name() == "sqlite"
}

// Snapshot - write a consistent copy of the database to path while it stays in use
func Snapshot(path string) error {
	if !UsingSQLite() {
		return ErrNotSQLite
	}
	return db.Exec("VACUUM INTO ?", path).Error
}

// Close - close the database, before its file is replaced
func Close() {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// Reopen - connect again after the file was replaced, bringing an older copy's tables up to date
func Reopen() {
	Start()
}
