package decision_repo
import ("context";"database/sql")
type Repo struct{db *sql.DB}
func New(db *sql.DB)*Repo{return &Repo{db}}
func (r *Repo)Migrate()error{
	ctx:=context.Background()
	_,e:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS decisions(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,title TEXT NOT NULL)")
	if e!=nil{return e}
	_,e=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS decision_votes(id TEXT PRIMARY KEY,decision_id TEXT NOT NULL,option TEXT NOT NULL)")
	return e
}
