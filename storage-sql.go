package ldbl

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// Base storage type for working with SQL databases.
// It wrapped around standart Go's database/sql interface.
// Supports base storage operations: Load(), Save(), Delete(), Select()
// and some SQL-related methods (see desc. of Query() method).
// Also supports transaction.
type SqlStorage struct {
	OptionalLogger
	db     *sql.DB
	logger *log.Logger
	tx     *sql.Tx
}

// Use this func for creating new instances of SQLStorage.
func NewSqlStorage(db *sql.DB) *SqlStorage {
	s := &SqlStorage{db: db}
	s.LogPrefix = "Storage"
	return s
}

func (s *SqlStorage) Save(item Storable) error {
	if item.Id() == 0 {
		return s.createNewEntry(item)
	}
	return s.updateEntry(item)
}

func (s *SqlStorage) Load(to Loadable, id uint64) error {
	sql := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s`=? LIMIT 1", to.CollectionName(), to.PKName())
	rows, columns, err := s.queryRows(sql, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		//TODO: Custom error type
		return fmt.Errorf("Entry %s#%d is not exists", to.CollectionName(), id)
	}
	return s.fillFromRow(rows, columns, to)
}

func (s *SqlStorage) Select(proto Loadable, results *[]Loadable, order Orderer, skip int, condition string, args ...interface{}) error {
	conditionSql := condition
	if conditionSql == "" {
		conditionSql = "1"
	}
	orderSql := ""
	if order != nil {
		orderSql = "ORDER BY " + order.OrderString()
	}
	limit := cap(*results)
	if limit == 0 {
		limit = -1
	}
	limitSql := ""
	if (skip > 0) || (limit > 0) {
		limitSql = fmt.Sprintf("LIMIT %d, %d", skip, limit)
	}
	sql := fmt.Sprintf(
		"SELECT * FROM `%s` WHERE %s %s %s",
		proto.CollectionName(),
		conditionSql,
		orderSql,
		limitSql)
	return s.loadByQuery(proto, sql, args, limit, results)
}

func (s *SqlStorage) Delete(item Loadable) error {
	if item.Id() == 0 {
		return nil
	}
	sql := fmt.Sprintf("DELETE FROM `%s` WHERE `%s`=%d", item.CollectionName(), item.PKName(), item.Id())
	_, err := s.exec(sql)
	item.Fill(0, nil)
	return err
}

//TODO: doc
func (s *SqlStorage) Query(builder SqlQueryBilder, results *[]Loadable) error {
	return s.loadByQuery(builder.ItemToLoad(), builder.Query(), builder.Args(), -1, results)
}

//TODO: doc
func (s *SqlStorage) Transaction(f func(t Transaction) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}
	s.Log("Transaction started")
	transaction := &SqlStorage{tx: tx, OptionalLogger: s.OptionalLogger}
	transaction.LogPrefix = "Storage (inside transaction)"
	err = f(transaction)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			//TODO: custom error type
			panic(fmt.Errorf(
				"Can't rollback a transaction: %s (which must be rolled back because of: %s)",
				rollbackErr.Error(),
				err.Error()))
		}
		s.Log("Transaction rolled back")
		return err
	}
	s.Log("Transaction will be commited")
	return tx.Commit()
}

func (s *SqlStorage) loadByQuery(proto Loadable, sql string, args []interface{}, limit int, results *[]Loadable) error {
	unlimited := limit == -1
	rows, columns, err := s.queryRows(sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for i := 0; rows.Next() && (unlimited || (i < limit)); i++ {
		clone := proto.Clone()
		if err := s.fillFromRow(rows, columns, clone); err != nil {
			return err
		}
		*results = append(*results, clone)
	}
	return nil
}

func (s *SqlStorage) fillFromRow(rows *sql.Rows, columns []string, to Loadable) error {
	if asStructured, ok := to.(Structured); ok {
		return s.fillFromRowStructured(rows, columns, asStructured)
	}
	columnsCnt := len(columns)
	ifaces := s.makeScanStrPlaceholders(columns, to)
	if err := rows.Scan(ifaces...); err != nil {
		//TODO: Custom error type
		return fmt.Errorf("%s: %s", to.CollectionName(), err.Error())
	}
	fields := make(map[string]interface{}, columnsCnt-1)
	id := uint64(0)
	for i := 0; i < columnsCnt; i++ {
		if columns[i] == to.PKName() {
			if idPtr, ok := ifaces[i].(*uint64); ok {
				id = *idPtr
			} else {
				//TODO: Custom error type
				return fmt.Errorf("%s: Primary key contains not integer value", to.CollectionName())
			}
			continue
		}
		fields[columns[i]] = *(ifaces[i].(*string))
	}
	return to.Fill(id, fields)
}

func (s *SqlStorage) fillFromRowStructured(rows *sql.Rows, columns []string, to Structured) error {
	ifaces, structFields := s.makeScanPlaceholders(columns, to)
	if err := rows.Scan(ifaces...); err != nil {
		//TODO: Custom error type
		return fmt.Errorf("%s: %s", to.CollectionName(), err.Error())
	}
	id := uint64(0)
	for i := 0; i < len(columns); i++ {
		if columns[i] == to.PKName() {
			if idPtr, ok := ifaces[i].(*uint64); ok {
				id = *idPtr
			} else {
				//TODO: Custom error type
				return fmt.Errorf("%s: Primary key contains not integer value", to.CollectionName())
			}
			continue
		}
		if _, present := structFields[columns[i]]; !present {
			continue
		}
		switch ifaces[i].(type) {
		case *int:
			structFields[columns[i]] = *(ifaces[i].(*int))
		case *int64:
			structFields[columns[i]] = *(ifaces[i].(*int64))
		case *float64:
			structFields[columns[i]] = *(ifaces[i].(*float64))
		case *bool:
			structFields[columns[i]] = *(ifaces[i].(*bool))
		case *[]byte:
			structFields[columns[i]] = *(ifaces[i].(*[]byte))
		case *time.Time:
			structFields[columns[i]] = *(ifaces[i].(*time.Time))
		case *string:
			structFields[columns[i]] = *(ifaces[i].(*string))
		case *uint64:
			structFields[columns[i]] = *(ifaces[i].(*uint64))
		}
	}
	return to.Fill(id, structFields)
}

func (s *SqlStorage) makeScanStrPlaceholders(columns []string, forValue Loadable) []interface{} {
	columnsCnt := len(columns)
	ifaces := make([]interface{}, columnsCnt)
	for i := 0; i < columnsCnt; i++ {
		if columns[i] == forValue.PKName() {
			ifaces[i] = new(uint64)
			continue
		}
		ifaces[i] = new(string)
	}
	return ifaces
}

func (s *SqlStorage) makeScanPlaceholders(columns []string, forValue Structured) ([]interface{}, map[string]interface{}) {
	columnsCnt := len(columns)
	ifaces := make([]interface{}, columnsCnt)
	structFields := forValue.FieldsStruct()
	for i := 0; i < columnsCnt; i++ {
		if columns[i] == forValue.PKName() {
			ifaces[i] = new(uint64)
			continue
		}
		if val, present := structFields[columns[i]]; present {
			switch val.(type) {
			case int:
				ifaces[i] = new(int)
			case int64:
				ifaces[i] = new(int64)
			case float64:
				ifaces[i] = new(float64)
			case bool:
				ifaces[i] = new(bool)
			case []byte:
				ifaces[i] = new([]byte)
			case time.Time:
				ifaces[i] = new(time.Time)
			case uint64:
				ifaces[i] = new(uint64)
			default: // trying to convert all other types to string
				ifaces[i] = new(string)
			}
			continue
		}
		ifaces[i] = new(string)
	}
	return ifaces, structFields
}

func (s *SqlStorage) createNewEntry(item Storable) error {
	sql, values := s.makeInsertSqlFor(item)
	res, err := s.exec(sql, values...)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	item.Fill(uint64(id), nil)
	return nil
}

func (s *SqlStorage) updateEntry(item Storable) error {
	fieldsCnt := len(item.Fields())
	fieldsSet := make([]string, 0, fieldsCnt)
	values := make([]interface{}, 0, fieldsCnt+1)
	for field, v := range item.Fields() {
		fieldsSet = append(fieldsSet, fmt.Sprintf("`%s`=?", field))
		values = append(values, v)
	}
	values = append(values, item.Id())
	setStr := strings.Join(fieldsSet, ",")
	sql := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s`=?", item.CollectionName(), setStr, item.PKName())
	_, err := s.exec(sql, values...)
	return err
}

func (s *SqlStorage) exec(sql string, values ...interface{}) (sql.Result, error) {
	s.Log("Executing '%s' with %d args", sql, len(values))
	if s.tx != nil {
		return s.tx.Exec(sql, values...)
	}
	return s.db.Exec(sql, values...)
}

func (s *SqlStorage) queryRows(sql string, values ...interface{}) (rows *sql.Rows, columns []string, err error) {
	s.Log("Executing '%s' with %d args", sql, len(values))
	if s.tx != nil {
		rows, err = s.tx.Query(sql, values...)
	} else {
		rows, err = s.db.Query(sql, values...)
	}
	if err != nil {
		return
	}
	if columns, err = rows.Columns(); err != nil {
		rows.Close()
	}
	return
}

func (s *SqlStorage) makeInsertSqlFor(item Storable) (string, []interface{}) {
	fieldsCnt := len(item.Fields())
	if fieldsCnt == 0 {
		return fmt.Sprintf("INSERT INTO `%s` values ()", item.CollectionName()), []interface{}{}
	}
	fields := make([]string, 0, fieldsCnt)
	placeholders := make([]string, 0, fieldsCnt)
	values := make([]interface{}, 0, fieldsCnt)
	for field, value := range item.Fields() {
		fields = append(fields, "`"+field+"`")
		values = append(values, value)
		placeholders = append(placeholders, "?")
	}
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) values (%s)",
		item.CollectionName(),
		strings.Join(fields, ","),
		strings.Join(placeholders, ","))
	return sql, values
}
