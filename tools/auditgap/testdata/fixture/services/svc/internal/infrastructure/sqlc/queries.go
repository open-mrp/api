package sqlc

type Queries struct{}

func (*Queries) CreateThing()         {}
func (*Queries) GetThing()            {}
func (*Queries) CreateOutboxMessage() {}
