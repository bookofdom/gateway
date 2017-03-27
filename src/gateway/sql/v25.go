package sql

func migrateToV25(db *DB) error {
	db.DisableSqliteTriggers()
	defer db.EnableSqliteTriggers()

	tx := db.MustBegin()
	tx.MustExec(db.SQL("v25/drop_accounts_stripe_payment_retry_attempt"))
	tx.MustExec(`UPDATE schema SET version = 25;`)
	return tx.Commit()
}
