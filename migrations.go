package ldbl

import (
	"database/sql"
	"fmt"
)

// Migration represents one migration step
type Migration struct {
	Up   string // Query to apply migration
	Down string // Query to rollback migration
}

// Migrator is a simple struct for applying migrations of DB schema.
// For the moment, it has extremely simple realisation and thus unassuming features.
// It's consist of list of Migration structs, that will be applyed one by one.
// Schema version is just a number of Migration in list.
type Migrator struct {
	migrations []Migration
	tabName    string
}

// Creates a new Migrator instance
func NewMigrator() *Migrator {
	return NewMigratorWithMigrations(make([]Migration, 0))
}

// Creates a new Migrator instance and inits it with given migration list
func NewMigratorWithMigrations(migrations []Migration) *Migrator {
	return &Migrator{migrations: migrations, tabName: "ldbl_migration"}
}

// Set name of service table, that stores current schema version
func (m *Migrator) SetMigrationTabName(n string) *Migrator {
	m.tabName = n
	return m
}

func (m *Migrator) AddMigration(migration Migration) {
	m.migrations = append(m.migrations, migration)
}

// Apply all updates to given DB adapter
func (m *Migrator) Update(db *sql.DB) error {
	currentVersion, _ := m.loadCurrentVersion(db)
	num := 0
	skipped := false
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for i, mig := range m.migrations {
		num = i + 1
		if !skipped && (num <= currentVersion) {
			continue
		} else {
			skipped = true
		}
		_, err := tx.Exec(mig.Up)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				panic(fmt.Errorf("Can't rollback transaction while migrating; DB may be broken"))
			}
			//TODO: custom error type
			return fmt.Errorf("Migration error: %s (Migration #%d: %s)", err.Error(), num, mig.Up)
		}
		currentVersion = num
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := m.storeCurrentVersion(db, currentVersion); err != nil {
		panic(fmt.Errorf("Can't store current version after transactions is applied"))
	}
	return nil
}

//TODO: implenent
func (m *Migrator) MigrateToVersion(db *sql.DB) error {
	panic(fmt.Errorf("Method not implemented"))
}

func (m *Migrator) createMigrTable(db *sql.DB) error {
	sql := fmt.Sprintf(
		"CREATE TABLE `%s` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `version` INTEGER NOT NULL, `performed_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);",
		m.tabName)
	_, err := db.Exec(sql)
	return err
}

func (m *Migrator) loadCurrentVersion(db *sql.DB) (int, error) {
	sql := fmt.Sprintf("SELECT `version` FROM `%s` WHERE 1 ORDER BY `id` DESC LIMIT 0, 1", m.tabName)
	rows, err := db.Query(sql)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, nil
	}
	id := 0
	if err := rows.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (m *Migrator) storeCurrentVersion(db *sql.DB, version int) error {
	if _, err := m.loadCurrentVersion(db); err != nil {
		if err := m.createMigrTable(db); err != nil {
			return err
		}
	}
	sql := fmt.Sprintf("INSERT INTO `%s` (`version`) VALUES (?)", m.tabName)
	_, err := db.Exec(sql, version)
	return err
}
