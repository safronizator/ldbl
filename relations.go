package ldbl

type RelationType int

const (
	HAS_ONE RelationType = iota
	HAS_MANY
	BELONGS_TO
)

type Relation struct {
	From              Loadable
	To                Loadable
	ForeignKey        string
	Type              RelationType
	GetForeignKeyFunc func() uint64
}

func NewHasOneRelation(from, to Loadable) *Relation {
	return &Relation{From: from, To: to, Type: HAS_ONE, ForeignKey: foreignKeyFor(from)}
}

func NewHasManyRelation(from, to Loadable) *Relation {
	return &Relation{From: from, To: to, Type: HAS_MANY, ForeignKey: foreignKeyFor(from)}
}

func NewBelongsToRelation(from, to Loadable) *Relation {
	return &Relation{From: from, To: to, Type: BELONGS_TO, ForeignKey: foreignKeyFor(to)}
}

//TODO: many-to-many: NewHasManyThroughRelation() ?

func (r *Relation) WithFK(fk string) *Relation {
	r.ForeignKey = fk
	return r
}

func (r *Relation) Reversed() *Relation {
	switch r.Type {
	case HAS_ONE, HAS_MANY:
		return &Relation{From: r.To, To: r.From, ForeignKey: r.ForeignKey, Type: BELONGS_TO}
	case BELONGS_TO:
		return &Relation{From: r.To, To: r.From, ForeignKey: r.ForeignKey, Type: HAS_MANY}
	}
	return nil
}

func foreignKeyFor(item Collectioned) string {
	//TODO: singularize?
	return item.CollectionName() + "_" + item.PKName()
}
