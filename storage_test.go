package ldbl_test

import (
	"ldbl"
	"strings"
	"testing"
)

var testFields = struct {
	filename string
	filesize uint64
}{
	filename: "test.jpg",
	filesize: 1024 * 800, // 800 KB
}

func provideStorage() ldbl.Storage {
	return ldbl.NewSqlStorage(provideTestDb())
}

func TestCreating(t *testing.T) {
	S := provideStorage()
	img := &Image{}
	img.SetField("filename", testFields.filename)
	img.SetField("filesize", testFields.filesize)
	img.SetField("users_id", 1)
	ok(t, S.Save(img))
	equals(t, uint64(1), img.Id())
}

func TestLoading(t *testing.T) {
	S := provideStorage()
	img := &Image{}
	ok(t, S.Load(img, 1))
	filename, ok := img.Field("filename").(string)
	assert(t, ok, "Can't convert value for field 'filename' (expected: string; actual: %T)", img.Field("filename"))
	equals(t, testFields.filename, filename)
	filesize, ok := img.Field("filesize").(uint64)
	assert(t, ok, "Can't convert value for field 'filesize' (expected: uint64; actual: %T)", img.Field("filesize"))
	equals(t, testFields.filesize, filesize)
}

func TestDeleting(t *testing.T) {
	S := provideStorage()
	img := &Image{}
	ok(t, S.Load(img, 1))
	ok(t, S.Delete(img))
	equals(t, uint64(0), img.Id())
	assert(t, S.Load(&Image{}, 1) != nil, "Loading of deleted item must return non-nil error")
}

func TestSelecting(t *testing.T) {
	// cleanup after previous tests
	removeTestDb()
	// and bootstraping some test data
	ok(t, makeTestData(provideTestDb()))
	S := provideStorage()

	var images []ldbl.Loadable

	// Selecting
	images = make([]ldbl.Loadable, 0)
	ok(t, S.Select(&Image{}, &images, nil, 0, ""))
	equals(t, TEST_IMAGES_CNT, len(images))
	for _, row := range images {
		_, ok := row.(*Image)
		assert(t, ok, "Got result of wrong type (expected: *Image; got: %T)", row)
	}

	// Conditions
	images = make([]ldbl.Loadable, 0)
	ok(t, S.Select(&Image{}, &images, nil, 0, "filename LIKE ?", "pig%"))
	for _, row := range images {
		img := row.(*Image)
		assert(t, strings.HasPrefix(img.Filename(), "pig"), "Filename of all selected images must start with 'pig' (got: %s)", img.Filename())
	}

	// Ordering
	images = make([]ldbl.Loadable, 0)
	ok(t, S.Select(&Image{}, &images, ldbl.OrderBy("id", ldbl.DESC), 0, ""))
	prev := uint64(100500)
	for _, row := range images {
		assert(t, row.Id() < prev, "Wrong ordering (current item id must be less than previous value of %d; got: %d)", prev, row.Id())
		prev = row.Id()
	}

	// Skipping
	images = make([]ldbl.Loadable, 0)
	skipItems := 2
	ok(t, S.Select(&Image{}, &images, ldbl.OrderBy("id", ldbl.DESC), skipItems, ""))
	equals(t, TEST_IMAGES_CNT-skipItems, len(images))

	// Limiting
	limit := 3
	images = make([]ldbl.Loadable, 0, limit) // capacity is the limit
	ok(t, S.Select(&Image{}, &images, nil, 0, ""))
	equals(t, limit, len(images))

	removeTestDb()
}
