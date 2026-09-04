package expense_repo
import ("context";"database/sql")
type Repo struct{db *sql.DB}
func New(db *sql.DB)*Repo{return &Repo{db}}
func (r *Repo)Migrate()error{
	ctx:=context.Background()
	_,e:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS expenses(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,category_id TEXT NOT NULL,title TEXT NOT NULL,amount INTEGER NOT NULL)")
	return e
}
