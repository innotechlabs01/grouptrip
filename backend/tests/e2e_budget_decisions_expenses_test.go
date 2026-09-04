package tests
import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"github.com/frg/grouptrip/internal/interfaces/http"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
	"database/sql"
	_ "modernc.org/sqlite"
)
func TestE2E_Budget_Decisions_Expenses(t *testing.T){
	db,_:=sql.Open("sqlite","file::memory:?cache=shared")
	repo:=fundrepo.New(db)
	s:=http.NewServerWithAuth(repo,nil,"",nil,nil,nil)
	go http.ListenAndServe(":8099",s)
	// create category
	body:=bytes.NewBufferString(`{"name":"Comida"}`)
	req,_:=http.NewRequest("POST","http://localhost:8099/budget/categories",body)
	req.Header.Set("Content-Type","application/json")
	req.Header.Set("Authorization","Bearer test")
	resp,_:=http.DefaultClient.Do(req)
	if resp.StatusCode!=http.StatusCreated{t.Fatalf("expected 201")}
	// list
	req,_=http.NewRequest("GET","http://localhost:8099/budget/categories",nil)
	req.Header.Set("Authorization","Bearer test")
	resp,_=http.DefaultClient.Do(req)
	var cats []map[string]string
	json.NewDecoder(resp.Body).Decode(&cats)
	if len(cats)==0{t.Fatalf("no categories")}
}
