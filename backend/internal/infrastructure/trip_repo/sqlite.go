package trip_repo
import ( "context"; "database/sql"; "fmt"; "github.com/frg/grouptrip/internal/domain/trip")
type SQLiteTripRepo struct{ db *sql.DB }
func New(db *sql.DB)*SQLiteTripRepo{ return &SQLiteTripRepo{db:db} }
func (r *SQLiteTripRepo) Migrate() error {
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS trips(id TEXT PRIMARY KEY,name TEXT NOT NULL,owner_id TEXT NOT NULL,created_at INTEGER NOT NULL)")
 if err!=nil{return err}
 _,err=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS trip_members(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,user_id TEXT NOT NULL,role TEXT NOT NULL,joined_at INTEGER NOT NULL)")
 return err
}
func (r *SQLiteTripRepo) CreateTrip(t *trip.Trip) error {
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"INSERT INTO trips(id,name,owner_id,created_at) VALUES(?,?,?,?)",t.ID,t.Name,t.OwnerID,t.CreatedAt.Unix())
 return err
}
func (r *SQLiteTripRepo) ListByOwner(ownerID string) ([]*trip.Trip,error){ return nil,nil }
