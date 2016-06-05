package ldbl_test

import (
	"ldbl"
	"strings"
	"testing"
)

var qParams = struct {
	filenameLike    string
	filesizeGreater uint64
	limit           int
	offset          int
}{
	filenameLike:    "kitty",
	filesizeGreater: 1024 * 100, // 100 KB
	limit:           2,
	offset:          1,
}

func provideSqlStorage() *ldbl.SqlStorage {
	return ldbl.NewSqlStorage(provideTestDb())
}

func TestQuerying(t *testing.T) {
	// cleanup after previous tests
	removeTestDb()
	// and bootstraping some test data
	ok(t, makeTestData(provideTestDb()))

	S := provideSqlStorage()

	// Parametered querying & Ordering
	results := make([]ldbl.Loadable, 0)
	testQuery := ldbl.
		Select(&Image{}).
		Where("images.filename LIKE ? AND images.filesize>?", qParams.filenameLike+"%", qParams.filesizeGreater).
		OrderBy("images.filesize", ldbl.ASC).
		OrderBy("images.created", ldbl.DESC)

	ok(t, S.Query(testQuery, &results))
	assert(t, len(results) > 0, "Got 0 results")
	prevSize := qParams.filesizeGreater
	for _, res := range results {
		img, isImg := res.(*Image)
		assert(t, isImg, "Got item of wrong type (expected: *Image; got: %T)", res)
		assert(t, strings.HasPrefix(img.Filename(), qParams.filenameLike), "'filename' must start from '%s'; got: '%s'", qParams.filenameLike, img.Filename())
		filesize := img.Filesize()
		assert(t, filesize > qParams.filesizeGreater, "Condition for 'filesize' failed: must me greater than %d; got: %d", qParams.filesizeGreater, filesize)
		assert(t, filesize >= prevSize, "Ordering failed: previous value of 'filesize' must be greater or equal than current (prev: %d; current: %d)", prevSize, filesize)
		prevSize = filesize
	}

	// Limiting
	results = make([]ldbl.Loadable, 0)
	ok(t, S.Query(testQuery.Limit(qParams.limit).Offset(qParams.offset), &results))
	equals(t, qParams.limit, len(results))

	removeTestDb()

}
