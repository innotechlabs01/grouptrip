package budget_repo
import ("context";"database/sql";"github.com/google/uuid")
type Repo struct{db *sql.DB}
func New(db *sql.DB)*Repo{return &Repo{db}}
func (r *Repo)Migrate()error{
	ctx:=context.Background()
	_,e:=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS budget_categories(id TEXT PRIMARY KEY,trip_id TEXT NOT NULL,name TEXT NOT NULL)")
	if e!=nil{return e}
	_,e=r.db.ExecContext(ctx,"CREATE TABLE IF NOT EXISTS budget_items(id TEXT PRIMARY KEY,category_id TEXT NOT NULL,name TEXT NOT NULL,amount INTEGER NOT NULL)")
	return e
}
type Category struct{ID, TripID, Name string}
type Item struct{ID, CategoryID, Name string; Amount int64}
func (r *Repo)ListCategories(ctx context.Context,tripID string)([]Category,error){...}
func (r *Repo)CreateCategory(ctx context.Context,c Category)error{...}
