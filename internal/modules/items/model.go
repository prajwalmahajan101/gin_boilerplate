package items

import (
	"encoding/json"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

// Item is the example domain model. It embeds store.BaseModel (id, is_active,
// timestamps) and maps to the items table via db tags for the generic repository.
type Item struct {
	store.BaseModel
	Name  string          `db:"name"`
	Code  string          `db:"code"`
	Notes json.RawMessage `db:"notes"`
}

func (Item) TableName() string { return "items" }
