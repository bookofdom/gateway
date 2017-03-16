package sql

func migrateToV26(db *DB) error {
	tx := db.MustBegin()
	tx.MustExec(db.SQL("v26/create_docker_images"))
	tx.MustExec(`UPDATE schema SET version = 26;`)
	return tx.Commit()
}
