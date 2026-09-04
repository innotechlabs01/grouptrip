package commands
import ("github.com/frg/grouptrip/internal/domain/trip"; "github.com/google/uuid")
type TripRepo interface{ CreateTrip(t *trip.Trip) error }
type CreateTripCmd struct{ Repo TripRepo }
type CreateTripInput struct{ Name string; OwnerID string }
func (c *CreateTripCmd) Execute(in CreateTripInput) (*trip.Trip, error){
 t,_:=trip.NewTrip(uuid.NewString(), in.Name, in.OwnerID)
 if err:=c.Repo.CreateTrip(t); err!=nil{ return nil, err }
 return t,nil
}
