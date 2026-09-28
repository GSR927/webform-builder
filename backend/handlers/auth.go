package handlers

import (
 "context"
 "crypto/rand"
 "crypto/sha256"
 "crypto/subtle"
 "database/sql"
 "encoding/hex"
 "encoding/json"
 "net/http"
 "os"
 "strings"
 "time"

 "golang.org/x/crypto/bcrypt"
)

type authContextKey struct{}
type Identity struct { UserID int64 `json:"id"`; TenantID int64 `json:"tenantId"`; Name string `json:"name"` }
type AuthHandler struct { DB *sql.DB }
func TenantFromContext(ctx context.Context) (int64,bool) {id,ok:=ctx.Value(authContextKey{}).(Identity);return id.TenantID,ok && id.TenantID>0}
func IdentityFromContext(ctx context.Context) (Identity,bool) {id,ok:=ctx.Value(authContextKey{}).(Identity);return id,ok && id.TenantID>0}
func randomToken()(string,error){b:=make([]byte,32);if _,err:=rand.Read(b);err!=nil{return "",err};return hex.EncodeToString(b),nil}
func cookieSecure()bool{return os.Getenv("COOKIE_SECURE")=="1"}
func makeCookie(name,value string,age int)*http.Cookie{return &http.Cookie{Name:name,Value:value,Path:"/",HttpOnly:name=="webform_session",Secure:cookieSecure(),SameSite:http.SameSiteStrictMode,MaxAge:age}}
func (h *AuthHandler) Login(w http.ResponseWriter,r *http.Request){
 var req struct{Email string `json:"email"`;Password string `json:"password"`}
 if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&req);err!=nil{http.Error(w,"invalid request",400);return}
 var id Identity;var hash string
 err:=h.DB.QueryRowContext(r.Context(),`SELECT id,tenant_id,name,password_hash FROM users WHERE email=?`,strings.ToLower(strings.TrimSpace(req.Email))).Scan(&id.UserID,&id.TenantID,&id.Name,&hash)
 if err!=nil || bcrypt.CompareHashAndPassword([]byte(hash),[]byte(req.Password))!=nil{writeJSON(w,401,map[string]string{"error":"invalid email or password"});return}
 token,err:=randomToken();if err!=nil{http.Error(w,"internal error",500);return};csrf,err:=randomToken();if err!=nil{http.Error(w,"internal error",500);return}
 td:=sha256.Sum256([]byte(token));cd:=sha256.Sum256([]byte(csrf))
 if _,err=h.DB.ExecContext(r.Context(),`INSERT INTO auth_sessions(token_hash,user_id,tenant_id,csrf_hash,expires_at) VALUES(?,?,?,?,?)`,td[:],id.UserID,id.TenantID,cd[:],time.Now().UTC().Add(8*time.Hour));err!=nil{http.Error(w,"login unavailable",500);return}
 http.SetCookie(w,makeCookie("webform_session",token,8*3600));http.SetCookie(w,makeCookie("webform_csrf",csrf,8*3600));w.Header().Set("Cache-Control","no-store");writeJSON(w,200,id)
}
func (h *AuthHandler) identity(r *http.Request)(Identity,[32]byte,bool){
 var zero [32]byte;cookie,err:=r.Cookie("webform_session");if err!=nil||len(cookie.Value)!=64{return Identity{},zero,false};token:=sha256.Sum256([]byte(cookie.Value));var id Identity;var csrf []byte
 err=h.DB.QueryRowContext(r.Context(),`SELECT s.user_id,s.tenant_id,u.name,s.csrf_hash FROM auth_sessions s JOIN users u ON u.id=s.user_id AND u.tenant_id=s.tenant_id WHERE s.token_hash=? AND s.expires_at > UTC_TIMESTAMP()`,token[:]).Scan(&id.UserID,&id.TenantID,&id.Name,&csrf)
 if err!=nil||len(csrf)!=32{return Identity{},zero,false};copy(zero[:],csrf);return id,zero,true
}
func (h *AuthHandler) Require(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){id,_,ok:=h.identity(r);if !ok{http.Error(w,"unauthorized",401);return};next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),authContextKey{},id)))})}
func (h *AuthHandler) CSRF(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 if r.Method==http.MethodGet||r.Method==http.MethodHead||r.Method==http.MethodOptions{next.ServeHTTP(w,r);return}
 _,expected,ok:=h.identity(r);if !ok{http.Error(w,"unauthorized",401);return};cookie,err:=r.Cookie("webform_csrf");if err!=nil||cookie.Value==""||r.Header.Get("X-CSRF-Token")!=cookie.Value{http.Error(w,"CSRF token required",403);return};got:=sha256.Sum256([]byte(cookie.Value));if subtle.ConstantTimeCompare(got[:],expected[:])!=1{http.Error(w,"invalid CSRF token",403);return};next.ServeHTTP(w,r)
})}
func (h *AuthHandler) Me(w http.ResponseWriter,r *http.Request){id,ok:=IdentityFromContext(r.Context());if !ok{http.Error(w,"unauthorized",401);return};w.Header().Set("Cache-Control","no-store");writeJSON(w,200,id)}
func (h *AuthHandler) Logout(w http.ResponseWriter,r *http.Request){if cookie,err:=r.Cookie("webform_session");err==nil{digest:=sha256.Sum256([]byte(cookie.Value));_,_=h.DB.ExecContext(r.Context(),`DELETE FROM auth_sessions WHERE token_hash=?`,digest[:])};http.SetCookie(w,makeCookie("webform_session","",-1));http.SetCookie(w,makeCookie("webform_csrf","",-1));w.WriteHeader(204)}
