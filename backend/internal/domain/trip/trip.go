package trip
import "time"
type Trip struct { ID string; Name string; OwnerID string; CreatedAt time.Time }
func NewTrip(id, name, ownerID string) (*Trip, error) {
 if id==""||name==""||ownerID=="" { return nil, error("invalid") }
 return &Trip{ID:id, Name:name, OwnerID:ownerID, CreatedAt:time.Now().UTC()}, nil
}
type TripMember struct { ID string; TripID string; UserID string; Role string; JoinedAt time.Time }
