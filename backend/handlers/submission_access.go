package handlers

import (
 "database/sql"
 "encoding/csv"
 "encoding/json"
 "errors"
 "fmt"
 "log"
 "net/http"
 "strconv"
 "strings"
 "time"

 "github.com/go-chi/chi/v5"
)

type AccessHandler struct{ DB *sql.DB }
type accessRow struct { ID string `json:"id"`; FormVersionID int64 `json:"formVersionId"`; SubmittedAt time.Time `json:"submittedAt"`; Payload map[string]any `json:"payload"` }
func (h *AccessHandler) ownedForm(r *http.Request) (int64,string,error) {
 id,err:=strconv.ParseInt(chi.URLParam(r,"id"),10,64);if err!=nil||id<=0{return 0,"",fmt.Errorf("invalid form id")}
 tenant,ok:=TenantFromContext(r.Context());if !ok{return 0,"",fmt.Errorf("missing identity")}
 var name string
 err=h.DB.QueryRowContext(r.Context(),`SELECT name FROM forms WHERE id=? AND tenant_id=?`,id,tenant).Scan(&name)
 if err!=nil{return 0,"",err};return id,name,nil
}
func accessError(w http.ResponseWriter,err error){if errors.Is(err,sql.ErrNoRows)||err.Error()=="invalid form id"{http.Error(w,"form not found",404)}else{log.Printf("submission access ownership error: %v",err);http.Error(w,"could not load submissions",500)}}
func escapeLike(s string)string{return strings.NewReplacer("\\","\\\\","%","\\%","_","\\_").Replace(s)}
func (h *AccessHandler) List(w http.ResponseWriter,r *http.Request) {
 id,name,err:=h.ownedForm(r);if err!=nil{accessError(w,err);return};tenant,_:=TenantFromContext(r.Context())
 page,_:=strconv.Atoi(r.URL.Query().Get("page"));if page<1{page=1};size,_:=strconv.Atoi(r.URL.Query().Get("pageSize"));if size<1{size=25};if size>100{size=100};if page>100000{http.Error(w,"page out of range",400);return}
 q:=strings.TrimSpace(r.URL.Query().Get("q"));if len(q)>120{http.Error(w,"query too long",400);return}
 where:=` WHERE tenant_id=? AND form_id=?`;args:=[]any{tenant,id};if q!=""{where+=` AND payload_json LIKE ?`;args=append(args,"%"+escapeLike(q)+"%")}
 var total int;if err=h.DB.QueryRowContext(r.Context(),`SELECT COUNT(*) FROM submissions`+where,args...).Scan(&total);err!=nil{log.Printf("submission count: %v",err);http.Error(w,"database error",500);return}
 queryArgs:=append(append([]any{},args...),size,(page-1)*size)
 rows,err:=h.DB.QueryContext(r.Context(),`SELECT id,form_version_id,submitted_at,payload_json FROM submissions`+where+` ORDER BY submitted_at DESC,id DESC LIMIT ? OFFSET ?`,queryArgs...);if err!=nil{log.Printf("submission list: %v",err);http.Error(w,"database error",500);return};defer rows.Close()
 items:=make([]accessRow,0);ids:=make([]int64,0);seen:=map[int64]bool{}
 for rows.Next(){var item accessRow;var raw []byte;if err=rows.Scan(&item.ID,&item.FormVersionID,&item.SubmittedAt,&raw);err!=nil{log.Printf("submission scan: %v",err);http.Error(w,"database error",500);return};if err=json.Unmarshal(raw,&item.Payload);err!=nil{log.Printf("submission JSON: %v",err);http.Error(w,"stored payload invalid",500);return};items=append(items,item);if !seen[item.FormVersionID]{seen[item.FormVersionID]=true;ids=append(ids,item.FormVersionID)}}
 if err=rows.Err();err!=nil{log.Printf("submission rows: %v",err);http.Error(w,"database error",500);return}
 versions:=map[string]any{};for _,versionID:=range ids{var number int;var raw []byte;err=h.DB.QueryRowContext(r.Context(),`SELECT version_number,schema_json FROM form_versions WHERE id=? AND tenant_id=? AND form_id=?`,versionID,tenant,id).Scan(&number,&raw);if err!=nil{log.Printf("submission version lookup: %v",err);continue};var schema map[string]any;if json.Unmarshal(raw,&schema)!=nil{continue};schema["versionNumber"]=number;versions[strconv.FormatInt(versionID,10)]=schema}
 w.Header().Set("Cache-Control","no-store");writeJSON(w,200,map[string]any{"items":items,"total":total,"page":page,"pageSize":size,"formName":name,"versionSchemas":versions})
}
func safeCSV(s string)string{if s!=""&&strings.ContainsRune("=+-@\t\r",rune(s[0])){return "'"+s};return s}
func (h *AccessHandler) Export(w http.ResponseWriter,r *http.Request){
 id,_,err:=h.ownedForm(r);if err!=nil{accessError(w,err);return};tenant,_:=TenantFromContext(r.Context());q:=strings.TrimSpace(r.URL.Query().Get("q"));if len(q)>120{http.Error(w,"query too long",400);return}
 where:=` WHERE tenant_id=? AND form_id=?`;args:=[]any{tenant,id};if q!=""{where+=` AND payload_json LIKE ?`;args=append(args,"%"+escapeLike(q)+"%")}
 rows,err:=h.DB.QueryContext(r.Context(),`SELECT id,form_version_id,submitted_at,payload_json FROM submissions`+where+` ORDER BY submitted_at DESC,id DESC`,args...);if err!=nil{http.Error(w,"database error",500);return};defer rows.Close()
 w.Header().Set("Content-Type","text/csv; charset=utf-8");w.Header().Set("Content-Disposition",fmt.Sprintf(`attachment; filename="form-%d-submissions.csv"`,id));w.Header().Set("Cache-Control","no-store")
 out:=csv.NewWriter(w);_ = out.Write([]string{"submission_id","form_version_id","submitted_at","payload_json"});for rows.Next(){var rowID string;var version int64;var at time.Time;var raw []byte;if err=rows.Scan(&rowID,&version,&at,&raw);err!=nil{return};_ = out.Write([]string{rowID,strconv.FormatInt(version,10),at.Format(time.RFC3339),safeCSV(string(raw))});out.Flush();if out.Error()!=nil{return}}
}
