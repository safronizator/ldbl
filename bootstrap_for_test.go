package ldbl_test

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"ldbl"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

var TEST_DB_NAME = "ldbl-sqlite_test.db"
var TEST_DB *sql.DB

var TEST_IMAGES_CNT = 9
var TEST_USERS_CNT = 2

//////// Helpful funcs

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		tb.Errorf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		tb.Errorf("\033[31m%s:%d: values are not equal: expected: %#v; got: %#v\033[39m", filepath.Base(file), line, exp, act)
	}
}

//////// DB bootstraping

func provideTestDb() *sql.DB {
	if TEST_DB != nil {
		return TEST_DB
	}
	db, err := createTestDb()
	if err != nil {
		panic(err)
	}
	TEST_DB = db
	return provideTestDb()
}

func provideCleanTestDb() *sql.DB {
	removeTestDb()
	return provideTestDb()
}

func cleanUp() {
	removeTestDb()
}

func removeTestDb() {
	if isFileExists(TEST_DB_NAME) {
		os.Remove(TEST_DB_NAME)
	}
	TEST_DB = nil
}

func createTestDb() (*sql.DB, error) {
	if isFileExists(TEST_DB_NAME) {
		if err := os.Remove(TEST_DB_NAME); err != nil {
			return nil, fmt.Errorf("Can't remove previous test data: %s", err.Error())
		}
	}
	db, err := sql.Open("sqlite3", TEST_DB_NAME)
	if err != nil {
		return nil, err
	}
	migr := ldbl.NewMigratorWithMigrations([]ldbl.Migration{
		ldbl.Migration{Up: `CREATE TABLE users (
	   		id INTEGER PRIMARY KEY AUTOINCREMENT,
	   		email VARCHAR(255) NOT NULL,
	   		created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);`},

		ldbl.Migration{Up: `CREATE TABLE images (
	   		id INTEGER PRIMARY KEY AUTOINCREMENT,
	   		users_id INTEGER NOT NULL,
	   		filename VARCHAR(255) NOT NULL,
	   		filesize INTEGER UNSIGNED NOT NULL DEFAULT '0',
	   		created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);`},

		ldbl.Migration{Up: `ALTER TABLE users ADD COLUMN images_cnt INTEGER NOT NULL DEFAULT '0';`},
	})
	if err := migr.Update(db); err != nil {
		return nil, err
	}
	return db, nil
}

func makeTestData(db *sql.DB) error {
	queries := []string{
		`INSERT INTO users (email) VALUES ('me@safron.su')`,
		`INSERT INTO users (email) VALUES ('alter-ego@gmail.com')`,

		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'kitty1.jpg', 46555)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'kitty2.jpg', 124899)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'kitty3.jpg', 164845)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'kitty4.jpg', 88190)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'kitty5.jpg', 164845)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'doggy1.jpg', 130229)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (1, 'doggy2.jpg', 440000)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (2, 'pig1.jpg', 898111)`,
		`INSERT INTO images (users_id, filename, filesize) VALUES (2, 'pig2.jpg', 800246)`,

		`UPDATE users SET images_cnt=(SELECT COUNT(*) FROM images WHERE images.users_id=users.id)`,
	}
	return dbExec(db, queries)
}

func dbExec(db *sql.DB, queries []string) error {
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func isFileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
