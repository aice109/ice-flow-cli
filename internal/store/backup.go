package store

// Backup writes a consistent snapshot of the database to path. It uses SQLite's
// VACUUM INTO, which produces a compact, fully-checkpointed copy without
// disturbing the open connection, so a running scheduler keeps working while
// the backup is taken. The destination must not already exist.
func (s *Store) Backup(path string) error {
	_, err := s.db.Exec(`VACUUM INTO ?`, path)
	return err
}
