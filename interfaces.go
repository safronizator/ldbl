// Package ldbl (aka "loadable") is a simple DB's data access & ORM lib for Go,
// that's not using reflection or any other "magic". Also, it's not forcing
// some pattern (as such as Active Record or something else), but gives ability to
// developer to choose the most suitable approach to work with data in DB (you can
// start with loading data rows to simple structures, with data fields represented
// in dicts, and, later, add more complicated logic - like custom struct fields,
// methods, event callbacks, etc.).
package ldbl

// Very base interface for items that could be loaded or stored to DB.
// It describes collection name (read "table name" for SQL databases)
// and name for the primary key field of item in the collection.
type Collectioned interface {
	PKName() string
	CollectionName() string
}

// Primary interface for items that could be loaded from DB.
// Id() must always return value for primary key or 0 in case of just created item
// Fill() will be called when item was loaded from DB, it's primary key value & fields map passed to func
// Clone() must return "empty" item of current concrete type. It's used when we need to fill items collection.
type Loadable interface {
	Collectioned
	Id() uint64
	Fill(id uint64, fields map[string]interface{}) error
	Clone() Loadable
}

// Primary interface for items that could be stored to DB.
// Fields() must return item's field values as map
type Storable interface {
	Loadable
	Fields() map[string]interface{}
}

// Iterface, implementing of which helps to describe items fields and it's types.
// FieldsStruct() must return fields map with it's initial values.
// Only fields returned by this method will be passed to save query.
type Structured interface {
	Loadable
	FieldsStruct() map[string]interface{}
}

// Describes universal method for getting field value by field name.
type FieldGetter interface {
	Field(name string) interface{}
}

// Describes universal method for setting field value by field name.
type FieldsSetter interface {
	SetField(name string, value interface{})
}

// Base storage interface. Every type that could request a DB, must implement it as minimum.
// Typically, high-level components has own handy methods for items selecting rather than base Select().
type Storage interface {
	Save(item Storable) error
	Load(to Loadable, id uint64) error
	Delete(item Loadable) error
	Select(proto Loadable, results *[]Loadable, order Orderer, skip int, condition string, args ...interface{}) error
}

// Just alias of Storage, needed for highlight cases when we need to use Transaction (instead of Storage) for write operations
type Transaction interface {
	Storage
}

// When DB supports transaction, related storage type will implement this interface.
type TransactionalStorage interface {
	Transaction(func(t Transaction) error) error
}
