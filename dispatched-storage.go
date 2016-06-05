package ldbl

import (
	"fmt"
	"log"
	"sync"
)

const (
	DELETE  = "delete"
	DELETED = "deleted"
	SAVE    = "save"
	SAVED   = "saved"
	UPDATE  = "update"
	UPDATED = "updated"
	CREATE  = "create"
	CREATED = "created"
)

// Describes trigger handler func
type Handler func(item Loadable, t Transaction) error

// Extended storage component.
// It wrapped around basic storage, and gives rich abilities for controlling all data manipulation processes,
// such as: triggering events; creating, managing and checking relations; caching items.
type DispatchedStorage struct {
	sync.RWMutex
	OptionalLogger
	storage         Storage
	relations       map[string]map[RelationType][]*Relation
	cache           *ItemsCache
	triggers        map[string][]Handler
	transactSupport bool
}

type TransactionWrapper struct {
	t Transaction
	s *DispatchedStorage
}

// Use this func for creating new instances of DispatchedStorage.
func NewDispatchedStorage(s Storage) *DispatchedStorage {
	_, transactSupport := s.(TransactionalStorage)
	ds := &DispatchedStorage{
		storage:         s,
		relations:       make(map[string]map[RelationType][]*Relation),
		cache:           NewItemsCache(100),
		triggers:        make(map[string][]Handler),
		transactSupport: transactSupport,
	}
	ds.LogPrefix = "Dispatcher"
	return ds
}

func (w *TransactionWrapper) Save(item Storable) error {
	return w.s.save(item, w)
}

func (w *TransactionWrapper) Delete(item Loadable) error {
	return w.s.delete(item, w)
}

func (w *TransactionWrapper) Load(to Loadable, id uint64) error {
	return w.t.Load(to, id)
}

func (w *TransactionWrapper) Select(proto Loadable, results *[]Loadable, order Orderer, skip int, condition string, args ...interface{}) error {
	return w.t.Select(proto, results, order, skip, condition, args...)
}

//TODO: doc
func (s *DispatchedStorage) SetCacheMaxSize(maxItemsCount int) *DispatchedStorage {
	s.cache = NewItemsCache(maxItemsCount)
	return s
}

func (s *DispatchedStorage) SetLogger(logger *log.Logger) *DispatchedStorage {
	s.OptionalLogger.SetLogger(logger)
	s.cache.SetLogger(logger)
	return s
}

func (s *DispatchedStorage) Save(item Storable) error {
	s.Lock()
	defer s.Unlock()
	return s.performWithTransaction(func(t Transaction) error {
		err := s.save(item, &TransactionWrapper{s: s, t: t})
		if err != nil {
			s.cache.Clear() //TODO: needs a better decision? (problem: if transaction was rolled back, it can make items in cache outdated)
		}
		return err
	})
}

func (s *DispatchedStorage) Delete(item Loadable) error {
	s.Lock()
	defer s.Unlock()
	return s.performWithTransaction(func(t Transaction) error {
		err := s.delete(item, &TransactionWrapper{s: s, t: t})
		if err != nil {
			s.cache.Clear() //TODO: needs a better decision? (problem: if transaction was rolled back, it can make items in cache outdated)
		}
		return err
	})
}

func (s *DispatchedStorage) Load(to Loadable, id uint64) error {
	if found := s.cache.Lookup(to, id); found {
		return nil
	}
	s.RLock()
	err := s.storage.Load(to, id)
	s.RUnlock()
	if err == nil {
		s.cache.Add(to)
	}
	return err
}

func (s *DispatchedStorage) Select(proto Loadable, results *[]Loadable, order Orderer, skip int, condition string, args ...interface{}) error {
	s.RLock()
	defer s.RUnlock()
	return s.storage.Select(proto, results, order, skip, condition, args...)
}

//TODO: doc
func (s *DispatchedStorage) RegisterRelation(relation *Relation) *DispatchedStorage {
	s.registerRelation(relation)
	if reversed := relation.Reversed(); reversed != nil {
		s.registerRelation(reversed)
	}
	return s
}

//TODO: doc
func (s *DispatchedStorage) RegisterHandler(forItem Collectioned, triggerName string, h Handler) *DispatchedStorage {
	fullName := forItem.CollectionName() + "." + triggerName
	if _, inited := s.triggers[fullName]; !inited {
		s.triggers[fullName] = make([]Handler, 0)
	}
	s.triggers[fullName] = append(s.triggers[fullName], h)
	s.Log("Handler added: %s.%s", forItem.CollectionName(), triggerName)
	return s
}

//TODO: doc
func (s *DispatchedStorage) PullTrigger(forItem Loadable, triggerName string) error {
	return s.pullTrigger(forItem, triggerName, nil)
}

//TODO: implement
//TODO: doc
func (s *DispatchedStorage) LoadSubitem(forItem, subitemProto Loadable, results *[]Loadable) error {
	rel := s.lookupRelationBetween(forItem, subitemProto, HAS_ONE)
	if rel == nil {
		//TODO: custom error type
		return fmt.Errorf(
			"No registered relation of type 'HAS_ONE' beetween '%s' & '%s'",
			forItem.CollectionName(),
			subitemProto.CollectionName())
	}
	//TODO: implement
	return nil
}

//TODO: doc
func (s *DispatchedStorage) LoadSubitems(forItem, subitemProto Loadable, results *[]Loadable) error {
	rel := s.lookupRelationBetween(forItem, subitemProto, HAS_MANY)
	if rel == nil {
		//TODO: custom error type
		return fmt.Errorf(
			"No registered relation of type 'HAS_MANY' beetween '%s' & '%s'",
			forItem.CollectionName(),
			subitemProto.CollectionName())
	}
	cond := fmt.Sprintf("`%s`.`%s`=?", rel.To.CollectionName(), rel.ForeignKey)
	return s.Select(subitemProto, results, nil, 0, cond, forItem.Id())
}

//TODO: doc
func (s *DispatchedStorage) LoadParentItem(forItem, parentItem Loadable) error {
	rel := s.lookupRelationBetween(forItem, parentItem, BELONGS_TO)
	if rel == nil {
		//TODO: custom error type
		return fmt.Errorf(
			"No registered relation of type 'BELONGS_TO' beetween '%s' & '%s'",
			forItem.CollectionName(),
			parentItem.CollectionName())
	}
	id, err := loadFkValue(forItem, rel)
	if err != nil {
		return err
	}
	return s.Load(parentItem, id)
}

func (s *DispatchedStorage) performWithTransaction(f func(t Transaction) error) error {
	if s.transactSupport {
		return s.storage.(TransactionalStorage).Transaction(f)
	}
	return f(s.storage.(Transaction))
}

func (s *DispatchedStorage) pullTrigger(forItem Loadable, triggerName string, t Transaction) error {
	fullName := forItem.CollectionName() + "." + triggerName
	s.Log("Trigger '%s' pulled", fullName)
	if _, inited := s.triggers[fullName]; !inited {
		return nil
	}
	transaction := t
	if transaction == nil {
		transaction = s.storage.(Transaction)
	}
	for _, handler := range s.triggers[fullName] {
		if err := handler(forItem, transaction); err != nil {
			return err
		}
	}
	return nil
}

func (s *DispatchedStorage) lookupRelationBetween(from, to Loadable, t RelationType) *Relation {
	cname := from.CollectionName()
	allRels, defined := s.relations[cname]
	if !defined {
		return nil
	}
	allRelsOfType, defined := allRels[t]
	if !defined {
		return nil
	}
	for _, rel := range allRelsOfType {
		if rel.To.CollectionName() == to.CollectionName() {
			return rel
		}
	}
	return nil
}

func (s *DispatchedStorage) registerRelation(relation *Relation) {
	cname := relation.From.CollectionName()
	if _, inited := s.relations[cname]; !inited {
		s.relations[cname] = make(map[RelationType][]*Relation)
	}
	if _, inited := s.relations[cname][relation.Type]; !inited {
		s.relations[cname][relation.Type] = make([]*Relation, 0, 3)
	}
	s.relations[cname][relation.Type] = append(s.relations[cname][relation.Type], relation)
	s.Log("Relation added (%d): %s --> %s", relation.Type, relation.From.CollectionName(), relation.To.CollectionName())
}

func (s *DispatchedStorage) save(item Storable, t *TransactionWrapper) error {
	preTrigger := CREATE
	postTrigger := CREATED
	if item.Id() > 0 {
		preTrigger = UPDATE
		postTrigger = UPDATED
	}
	if err := s.checkRelated(item); err != nil {
		return err
	}
	if err := s.pullTrigger(item, SAVE, t); err != nil {
		return err
	}
	if err := s.pullTrigger(item, preTrigger, t); err != nil {
		return err
	}
	if err := t.t.Save(item); err != nil {
		return nil
	}
	if err := s.pullTrigger(item, postTrigger, t); err != nil {
		return err
	}
	if err := s.pullTrigger(item, SAVED, t); err != nil {
		return err
	}
	s.cache.Add(item)
	return nil
}

func (s *DispatchedStorage) delete(item Loadable, t *TransactionWrapper) error {
	if err := s.pullTrigger(item, DELETE, t); err != nil {
		return err
	}
	if err := s.deleteRelated(item, t); err != nil {
		return err
	}
	s.cache.Remove(item)
	if err := t.t.Delete(item); err != nil {
		return err
	}
	if err := s.pullTrigger(item, DELETED, t); err != nil {
		return err
	}
	return nil
}

func (s *DispatchedStorage) deleteRelated(forItem Loadable, t *TransactionWrapper) error {
	rels := s.getRelationsOfType(forItem, HAS_MANY)
	//TODO: do for HAS_ONE
	if rels == nil {
		return nil
	}
	for _, rel := range rels {
		results := make([]Loadable, 0)
		cond := fmt.Sprintf("`%s`.`%s`=?", rel.To.CollectionName(), rel.ForeignKey)
		if err := s.storage.Select(rel.To, &results, nil, 0, cond, forItem.Id()); err != nil {
			return err
		}
		for _, subitem := range results {
			if err := s.delete(subitem, t); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *DispatchedStorage) checkRelated(forItem Loadable) error {
	rels := s.getRelationsOfType(forItem, BELONGS_TO)
	if rels == nil {
		return nil
	}
	var id uint64
	var err error
	for _, rel := range rels {
		id, err = loadFkValue(forItem, rel)
		if err != nil {
			return err
		}
		err = s.storage.Load(rel.To, id)
		if err != nil {
			//TODO: Custom error type
			return fmt.Errorf(
				"Can't load related item %s#%d, which linked in %s.%s",
				rel.To.CollectionName(),
				id,
				forItem.CollectionName(),
				rel.ForeignKey)
		}
	}
	return nil
}

func (s *DispatchedStorage) getRelationsOfType(forItem Loadable, t RelationType) []*Relation {
	rels, _ := s.relations[forItem.CollectionName()][t]
	return rels
}

func loadFkValue(forItem Loadable, rel *Relation) (uint64, error) {
	var id uint64
	var gotId bool
	if rel.GetForeignKeyFunc != nil {
		id = rel.GetForeignKeyFunc()
	} else if getter, isGetter := forItem.(FieldGetter); isGetter {
		rawId := getter.Field(rel.ForeignKey)
		if id, gotId = uint64Value(rawId); !gotId {
			//TODO: Custom error type
			return 0, fmt.Errorf("Foreign key %s.%s contains not uint64 value (%v)", forItem.CollectionName(), rel.ForeignKey, rawId)
		}
	} else {
		//TODO: Custom error type
		return 0, fmt.Errorf("Can't check related item of '%s' collection (when processing '%s')", forItem.CollectionName(), rel.To.CollectionName())
	}
	return id, nil
}

func uint64Value(from interface{}) (v uint64, got bool) {
	switch from.(type) {
	case uint64:
		return from.(uint64), true
	case uint:
		return uint64(from.(uint)), true
	case int:
		return uint64(from.(int)), true
	}
	return 0, false
}

//////////
//////////
//////////
//////////
//////////
//////////
//////////
//////////
