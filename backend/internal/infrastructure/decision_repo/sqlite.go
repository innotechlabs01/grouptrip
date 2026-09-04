package decision_repo
import "context"; "database/sql"
type Repo struct{ db *sql.DB }
func New(db *sql.DB)*Repo{ return &Repo{db} }
func (r *Repo) Migrate() error {
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS decisions(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,question TEXT NOT NULL)")
 if err!=nil{return err}
 _,err=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS votes(id TEXT PRIMARY KEY,decision_id TEXT NOT NULL,user_id TEXT NOT NULL,option TEXT NOT NULL)")
 return err
}
