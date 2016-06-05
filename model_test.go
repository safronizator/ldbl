package ldbl_test

import (
	"ldbl"
	"testing"
	"time"
)

////// Test model

type User struct {
	id          uint64
	Email       string
	Created     time.Time
	ImagesCount int
}

type Image struct {
	ldbl.Model
}

func (u *User) PKName() string {
	return "id"
}
func (u *User) CollectionName() string {
	return "users"
}
func (u *User) Id() uint64 {
	return u.id
}
func (u *User) Fill(id uint64, fields map[string]interface{}) error {
	u.id = id
	u.Email, _ = fields["email"].(string)
	u.Created, _ = fields["created"].(time.Time)
	u.ImagesCount, _ = fields["images_cnt"].(int)
	return nil
}
func (u *User) Clone() ldbl.Loadable {
	return &User{Email: u.Email, Created: u.Created}
}
func (u *User) Fields() map[string]interface{} {
	return map[string]interface{}{
		"email":      u.Email,
		"created":    u.Created,
		"images_cnt": u.ImagesCount,
	}
}
func (u *User) FieldsStruct() map[string]interface{} {
	return u.Fields()
}

func (i *Image) Filename() string {
	return i.Field("filename").(string)
}
func (i *Image) Filesize() uint64 {
	return i.Field("filesize").(uint64)
}
func (i *Image) Created() time.Time {
	return i.Field("created").(time.Time)
}
func (i *Image) CollectionName() string {
	return "images"
}
func (i *Image) Clone() ldbl.Loadable {
	return &Image{i.Model.Clone()}
}
func (i *Image) FieldsStruct() map[string]interface{} {
	return map[string]interface{}{
		"users_id": uint64(0),
		"filename": "",
		"filesize": uint64(0),
		"created":  time.Now(),
	}
}
func (i *Image) SetUser(u *User) {
	i.SetField("users_id", u.Id())
}

func (i *Image) LoadUser(s *ldbl.DispatchedStorage) (*User, error) {
	user := &User{}
	if err := s.LoadParentItem(i, user); err != nil {
		return nil, err
	}
	return user, nil
}

////// Model tests

func TestSetField(t *testing.T) {
	img := &Image{}
	img.SetField("filesize", uint64(100*1024))
	got := img.Field("filesize")
	asUint64, ok := got.(uint64)
	assert(t, ok, "Can't get back value set for field 'filesize' (got value of wrong type; expected: uint64; got: %T)", got)
	equals(t, uint64(100*1024), asUint64)
}
