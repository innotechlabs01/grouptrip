package e2e
import (
  "bytes"
  "encoding/json"
  "net/http"
  "testing"
)
func TestAuthAndTripsE2E(t *testing.T){
  base := "http://localhost:8080"
  payload := map[string]string{"email":"test@example.com","password":"Pass123!"}
  b,_ := json.Marshal(payload)
  // register
  req,_:=http.NewRequest("POST",base+"/auth/register",bytes.NewReader(b))
  req.Header.Set("Content-Type","application/json")
  res,_:=http.DefaultClient.Do(req)
  if res.StatusCode!=http.StatusCreated{ t.Fatalf("register failed %d",res.StatusCode) }
  // login
  req,_=http.NewRequest("POST",base+"/auth/login",bytes.NewReader(b))
  req.Header.Set("Content-Type","application/json")
  res,_=http.DefaultClient.Do(req)
  if res.StatusCode!=http.StatusOK{ t.Fatalf("login failed %d",res.StatusCode) }
  // create trip
  tripPayload:=map[string]string{"title":"Trip Test"}
  b,_=json.Marshal(tripPayload)
  req,_=http.NewRequest("POST",base+"/trips",bytes.NewReader(b))
  req.Header.Set("Content-Type","application/json")
  res,_=http.DefaultClient.Do(req)
  if res.StatusCode!=http.StatusCreated{ t.Fatalf("create trip failed %d",res.StatusCode) }
}
