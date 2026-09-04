package trip_repo
import (
 "context"
 "database/sql"
 "time"
 "github.com/frg/grouptrip/internal/domain/trip"
)
type SQLiteTripRepo struct{ db *sql.DB }
func New(db *sql.DB)*SQLiteTripRepo{ return &SQLiteTripRepo{db:db} }
func (r *SQLiteTripRepo) Migrate() error {
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS trips(id TEXT PRIMARY KEY,name TEXT NOT NULL,owner_id TEXT NOT NULL,created_at INTEGER NOT NULL)")
 if err!=nil{return err}
 _,err=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS trip_members(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,user_id TEXT NOT NULL,role TEXT NOT NULL,joined_at INTEGER NOT NULL, UNIQUE(trip_id,user_id))")
 return err
}
func (r *SQLiteTripRepo) CreateTrip(t *trip.Trip) error {
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"INSERT INTO trips(id,name,owner_id,created_at) VALUES(?,?,?,?)",t.ID,t.Name,t.OwnerID,t.CreatedAt.Unix())
 return err
}
func (r *SQLiteTripRepo) ListByOwner(ownerID string) ([]*trip.Trip, error){
 ctx:=context.Background()
 rows,err:=r.db.QueryContext(ctx,"SELECT id,name,owner_id,created_at FROM trips WHERE owner_id=?",ownerID)
 if err!=nil{return nil,err}
 defer rows.Close()
 var out []*trip.Trip
 for rows.Next(){
  var id,name,owner string; var ts int64
  rows.Scan(&id,&name,&owner,&ts)
  out=append(out,&trip.Trip{ID:id,Name:name,OwnerID:owner,CreatedAt:time.Unix(ts,0).UTC()})
 }
 return out,nil
}
func (r *SQLiteTripRepo) AddMember(tripID,userID,role string) error{
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"INSERT INTO trip_members(id,trip_id,user_id,role,joined_at) VALUES(?,?,?,?,?)", "m"+tripID+userID, tripID,userID,role,time.Now().UTC().Unix())
 return err
}
