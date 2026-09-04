package budget
type Category struct{ ID string; TripID string; Name string }
type Item struct{ ID string; CategoryID string; Name string; Amount int64 }
func NewCategory(id,tripID,name string) *Category { return &Category{ID:id,TripID:tripID,Name:name} }
