package ldbl

type Model struct {
	id     uint64
	fields map[string]interface{}
}

func (m *Model) PKName() string {
	return "id"
}

func (m *Model) Fill(id uint64, fields map[string]interface{}) error {
	m.id = id
	if fields != nil {
		m.fields = fields
	}
	return nil
}

func (m *Model) Id() uint64 {
	return m.id
}

func (m *Model) Fields() map[string]interface{} {
	return m.fields
}

func (m *Model) Field(name string) interface{} {
	if m.fields == nil {
		return nil
	}
	return m.fields[name]
}

func (m *Model) SetField(name string, value interface{}) {
	if m.fields == nil {
		m.fields = make(map[string]interface{})
	}
	m.fields[name] = value
}

func (m *Model) Clone() Model {
	return Model{fields: m.fields}
}
