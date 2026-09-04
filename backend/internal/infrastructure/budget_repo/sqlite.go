package budget_repo
import "context"; "database/sql"
type Repo struct{ db *sql.DB }
func New(db *sql.DB)*Repo{ return &Repo{db} }
func (r *Repo) Migrate() error {
 ctx:=context.Background()
 _,err:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS budget_categories(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,name TEXT NOT NULL)")
 if err!=nil{return err}
 _,err=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS budget_items(id TEXT PRIMARY KEY,category_id TEXT NOT NULL,name TEXT NOT NULL,amount INTEGER NOT NULL)")
 return err
}
