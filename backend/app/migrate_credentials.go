package app

import (
	"database/sql"
	"fmt"
)

// migrateCredentialsToKeyring copies passwords and tokens from the database to the keyring
// This runs BEFORE SQL migrations that drop the password columns
// TODO: Remove this file after all users have migrated (e.g., v1.x release)
func (a *App) migrateCredentialsToKeyring(db *sql.DB) error {
	if a.keyring == nil {
		return fmt.Errorf("keyring not initialized")
	}

	// Check if password column still exists in repositories table (idempotent check)
	hasPasswordColumn, err := a.columnExists(db, "repositories", "password")
	if err != nil {
		return fmt.Errorf("failed to check for password column: %w", err)
	}

	if hasPasswordColumn {
		// Migrate repository passwords
		if err := a.migrateRepositoryPasswords(db); err != nil {
			a.log.Warnf("Failed to migrate repository passwords: %v", err)
		}
	}

	// Check if access_token column still exists in users table
	hasAccessTokenColumn, err := a.columnExists(db, "users", "access_token")
	if err != nil {
		return fmt.Errorf("failed to check for access_token column: %w", err)
	}

	if hasAccessTokenColumn {
		// Migrate auth tokens
		if err := a.migrateAuthTokens(db); err != nil {
			a.log.Warnf("Failed to migrate auth tokens: %v", err)
		}
	}

	return nil
}

// columnExists checks if a column exists in a table
func (a *App) columnExists(db *sql.DB, table, column string) (bool, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := db.Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// migrateRepositoryPasswords copies repository passwords from the database to the keyring
func (a *App) migrateRepositoryPasswords(db *sql.DB) error {
	rows, err := db.Query("SELECT id, password FROM repositories WHERE password IS NOT NULL AND password != ''")
	if err != nil {
		return fmt.Errorf("failed to query repositories: %w", err)
	}
	defer rows.Close()

	var migrated, skipped int
	for rows.Next() {
		var id int
		var password string
		if err := rows.Scan(&id, &password); err != nil {
			a.log.Warnf("Failed to scan repository row: %v", err)
			continue
		}

		// Check if already in keyring
		if a.keyring.HasRepositoryPassword(id) {
			skipped++
			continue
		}

		// Store in keyring
		if err := a.keyring.SetRepositoryPassword(id, password); err != nil {
			a.log.Warnf("Failed to migrate password for repository %d: %v", id, err)
			continue
		}

		// Update has_password flag in database
		if _, err := db.Exec("UPDATE repositories SET has_password = true WHERE id = ?", id); err != nil {
			a.log.Warnf("Failed to update has_password flag for repository %d: %v", id, err)
		}

		migrated++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating repository rows: %w", err)
	}

	if migrated > 0 {
		a.log.Infof("Migrated %d repository passwords to keyring (skipped %d already migrated)", migrated, skipped)
	}
	return nil
}

// migrateAuthTokens copies auth tokens from the database to the keyring
func (a *App) migrateAuthTokens(db *sql.DB) error {
	rows, err := db.Query("SELECT access_token, refresh_token FROM users WHERE access_token IS NOT NULL AND access_token != ''")
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var accessToken, refreshToken sql.NullString
		if err := rows.Scan(&accessToken, &refreshToken); err != nil {
			a.log.Warnf("Failed to scan user row: %v", err)
			continue
		}

		// Check if already in keyring
		if _, err := a.keyring.GetAccessToken(); err == nil {
			a.log.Debug("Auth tokens already in keyring, skipping migration")
			continue
		}

		// Store tokens in keyring
		if accessToken.Valid && accessToken.String != "" {
			if err := a.keyring.SetAccessToken(accessToken.String); err != nil {
				a.log.Warnf("Failed to migrate access token: %v", err)
				continue
			}
		}

		if refreshToken.Valid && refreshToken.String != "" {
			if err := a.keyring.SetRefreshToken(refreshToken.String); err != nil {
				a.log.Warnf("Failed to migrate refresh token: %v", err)
				continue
			}
		}

		a.log.Info("Migrated auth tokens to keyring")
	}

	return rows.Err()
}
