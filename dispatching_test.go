package ldbl_test

import (
	"fmt"
	"ldbl"
	// "log"
	// "os"
	"testing"
)

type triggerCheck struct {
	SAVE    bool
	SAVED   bool
	UPDATE  bool
	UPDATED bool
	CREATE  bool
	CREATED bool
	DELETE  bool
	DELETED bool
}

func provideDispatchedStorage() *ldbl.DispatchedStorage {
	sqlStrg := provideSqlStorage()
	s := ldbl.NewDispatchedStorage(sqlStrg)
	// logger := log.New(os.Stderr, "", log.LstdFlags)
	// sqlStrg.SetLogger(logger)
	// s.SetLogger(logger)
	s.RegisterRelation(ldbl.NewHasManyRelation(&User{}, &Image{}))
	return s
}

func initTriggers(s *ldbl.DispatchedStorage) *triggerCheck {
	s.RegisterHandler(&Image{}, ldbl.CREATED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		user, err := i.(*Image).LoadUser(s)
		if err != nil {
			return err
		}
		user.ImagesCount = user.ImagesCount + 1
		return tx.Save(user)
	})
	s.RegisterHandler(&Image{}, ldbl.DELETED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		user, err := i.(*Image).LoadUser(s)
		if err != nil {
			return err
		}
		user.ImagesCount = user.ImagesCount - 1
		return tx.Save(user)
	})
	t := &triggerCheck{}
	s.RegisterHandler(&User{}, ldbl.SAVE, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.SAVE = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.SAVED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.SAVED = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.UPDATE, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.UPDATE = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.UPDATED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.UPDATED = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.CREATE, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.CREATE = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.CREATED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.CREATED = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.DELETE, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.DELETE = true
		return nil
	})
	s.RegisterHandler(&User{}, ldbl.DELETED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		t.DELETED = true
		return nil
	})
	return t
}

func TestRelations(t *testing.T) {
	// cleanup after previous tests
	removeTestDb()
	// and bootstraping some test data
	ok(t, makeTestData(provideTestDb()))

	S := provideDispatchedStorage()

	// "Has many" & "Belongs To" - related items loading
	user := &User{}
	ok(t, S.Load(user, 1))
	userImages := make([]ldbl.Loadable, 0)
	ok(t, S.LoadSubitems(user, &Image{}, &userImages))
	assert(t, len(userImages) > 0, "Can't load user's images")
	for _, item := range userImages {
		img, isImg := item.(*Image)
		assert(t, isImg, "Got item of wrong type: expected: *Image; got: %T", item)
		imgUser, err := img.LoadUser(S)
		ok(t, err)
		equals(t, user, imgUser)
	}

	// Cascade deleting
	ok(t, S.Delete(user))
	userImages = make([]ldbl.Loadable, 0)
	ok(t, S.LoadSubitems(user, &Image{}, &userImages))
	equals(t, 0, len(userImages))

	// Relations check on create
	img := &Image{}
	img.SetField("filename", "test.jpg")
	img.SetField("users_id", uint64(100500))
	err := S.Save(img)
	//TODO: error type check
	assert(t, err != nil, "Got nil error when trying to save *Image with incorrect foreign key")

	removeTestDb()
}

func TestTriggering(t *testing.T) {
	// cleanup after previous tests
	removeTestDb()
	// and bootstraping some test data
	ok(t, makeTestData(provideTestDb()))

	S := provideDispatchedStorage()
	check := initTriggers(S)

	// Check that all triggers are called
	user := &User{}
	user.Email = "tester@test.com"
	ok(t, S.Save(user))
	assert(t, check.SAVE, "SAVE trigger was not pulled")
	assert(t, check.SAVED, "SAVED trigger was not pulled")
	assert(t, check.CREATE, "CREATE trigger was not pulled")
	assert(t, check.CREATED, "CREATE trigger was not pulled")
	check.SAVE = false
	check.SAVED = false
	user.Email = "changed@test.com"
	ok(t, S.Save(user))
	assert(t, check.SAVE, "SAVE trigger was not pulled")
	assert(t, check.SAVED, "SAVED trigger was not pulled")
	assert(t, check.UPDATE, "UPDATE trigger was not pulled")
	assert(t, check.UPDATED, "UPDATED trigger was not pulled")
	ok(t, S.Delete(user))
	assert(t, check.DELETE, "DELETE trigger was not pulled")
	assert(t, check.DELETED, "DELETED trigger was not pulled")

	// Counters updating on DELETE/CREATE
	user = &User{}
	testUserId := uint64(1)
	ok(t, S.Load(user, testUserId))
	imagesCount := user.ImagesCount
	img := &Image{}
	img.SetField("filename", "test.jpg")
	img.SetField("users_id", testUserId)
	ok(t, S.Save(img))
	ok(t, S.Load(user, testUserId)) // user needs to be updated
	equals(t, imagesCount+1, user.ImagesCount)
	ok(t, S.Delete(img))
	ok(t, S.Load(user, testUserId)) // user needs to be updated
	equals(t, imagesCount, user.ImagesCount)

	removeTestDb()
}

func TestTransactions(t *testing.T) {
	// cleanup after previous tests
	removeTestDb()
	// and bootstraping some test data
	ok(t, makeTestData(provideTestDb()))

	S := provideDispatchedStorage()
	initTriggers(S)
	S.RegisterHandler(&Image{}, ldbl.DELETED, func(i ldbl.Loadable, tx ldbl.Transaction) error {
		return fmt.Errorf("This is a test error that should lead to rollback a transaction")
	})
	user := &User{}
	testUserId := uint64(1)
	ok(t, S.Load(user, testUserId))
	imagesCount := user.ImagesCount
	userImages := make([]ldbl.Loadable, 0)
	ok(t, S.LoadSubitems(user, &Image{}, &userImages))
	firstImage := userImages[0].(*Image)
	imgId := firstImage.Id()
	err := S.Delete(firstImage)
	assert(t, err != nil, "Error returned from trigger must be returned from delete operation")
	ok(t, S.Load(user, testUserId))          // user reloading
	equals(t, imagesCount, user.ImagesCount) // Counter was not updated
	ok(t, S.Load(firstImage, imgId))         // Image remained in DB
	removeTestDb()
}
