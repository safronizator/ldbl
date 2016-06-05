package ldbl_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	cleanUp()
	os.Exit(code)
}
