package http
import "sync"
var (
	budgetCatsMu sync.Mutex; budgetCats []map[string]string
	budgetItemsMu sync.Mutex; budgetItems []map[string]interface{}
	decisionsMu sync.Mutex; decisions []map[string]interface{}
	votesMu sync.Mutex; votes []map[string]string
	expensesMu sync.Mutex; expenses []map[string]interface{}
	tripsMu sync.Mutex; trips []map[string]string
)
func init(){
	budgetCats=[]map[string]string{}; budgetItems=[]map[string]interface{}{}
	decisions=[]map[string]interface{}{}; votes=[]map[string]string{}
	expenses=[]map[string]interface{}{}; trips=[]map[string]string{}
}
