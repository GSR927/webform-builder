package handlers

import (
 "database/sql"
 "encoding/json"
 "errors"
 "net/http"
 "strconv"
 "time"

 "github.com/go-chi/chi/v5"
 "github.com/example/webform-builder/backend/models"
)

type FormsHandler struct{DB *sql.DB}
func formID(r *http.Request)(int64,error){return strconv.ParseInt(chi.URLParam(r,"id"),10,64)}
func (h *FormsHandler) CreateForm(w http.ResponseWriter,r *http.Request){
 tenant,ok:=TenantFromContext(r.Context());if !ok{http.Error(w,"unauthorized",401);return}
 var req struct{Name string `json:"name"`};if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20)).Decode(&req);err!=nil||req.Name==""{http.Error(w,"invalid name",400);return}
 schema,_:=json.Marshal(models.FormSchema{Fields:[]models.FieldDef{}})
 result,err:=h.DB.ExecContext(r.Context(),`INSERT INTO forms(tenant_id,name,status,draft_schema_json,created_at,updated_at) VALUES(?,?,'draft',?,?,?)`,tenant,req.Name,schema,time.Now(),time.Now());if err!=nil{http.Error(w,"create failed",500);return};id,_:=result.LastInsertId();writeJSON(w,201,map[string]any{"id":id})
}
func (h *FormsHandler) GetDraft(w http.ResponseWriter,r *http.Request){
 tenant,ok:=TenantFromContext(r.Context());if !ok{http.Error(w,"unauthorized",401);return};id,err:=formID(r);if err!=nil{http.Error(w,"invalid id",400);return}
 var name string;var schema []byte;var version sql.NullInt64
 err=h.DB.QueryRowContext(r.Context(),`SELECT name,draft_schema_json,current_version_id FROM forms WHERE id=? AND tenant_id=?`,id,tenant).Scan(&name,&schema,&version)
 if errors.Is(err,sql.ErrNoRows){http.Error(w,"form not found",404);return};if err!=nil{http.Error(w,"load failed",500);return}
 response:=map[string]any{"formId":id,"name":name,"schema":json.RawMessage(schema)};if version.Valid{response["publishedVersionId"]=version.Int64};w.Header().Set("Cache-Control","no-store");writeJSON(w,200,response)
}
func (h *FormsHandler) UpdateDraft(w http.ResponseWriter,r *http.Request){
 tenant,ok:=TenantFromContext(r.Context());if !ok{http.Error(w,"unauthorized",401);return};id,err:=formID(r);if err!=nil{http.Error(w,"invalid id",400);return}
 var schema models.FormSchema;if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20)).Decode(&schema);err!=nil||schema.Fields==nil{http.Error(w,"invalid schema",400);return};raw,_:=json.Marshal(schema)
 result,err:=h.DB.ExecContext(r.Context(),`UPDATE forms SET draft_schema_json=?,updated_at=? WHERE id=? AND tenant_id=?`,raw,time.Now(),id,tenant);if err!=nil{http.Error(w,"save failed",500);return};affected,err:=result.RowsAffected();if err!=nil{http.Error(w,"save failed",500);return};if affected==0{http.Error(w,"form not found",404);return};writeJSON(w,200,map[string]string{"status":"saved"})
}
func (h *FormsHandler) PublishForm(w http.ResponseWriter,r *http.Request){
 tenant,ok:=TenantFromContext(r.Context());if !ok{http.Error(w,"unauthorized",401);return};id,err:=formID(r);if err!=nil{http.Error(w,"invalid id",400);return}
 tx,err:=h.DB.BeginTx(r.Context(),nil);if err!=nil{http.Error(w,"db error",500);return};defer tx.Rollback()
 var schema []byte;err=tx.QueryRowContext(r.Context(),`SELECT draft_schema_json FROM forms WHERE id=? AND tenant_id=? FOR UPDATE`,id,tenant).Scan(&schema);if errors.Is(err,sql.ErrNoRows){http.Error(w,"form not found",404);return};if err!=nil{http.Error(w,"db error",500);return}
 var n int;err=tx.QueryRowContext(r.Context(),`SELECT COALESCE(MAX(version_number),0)+1 FROM form_versions WHERE form_id=?`,id).Scan(&n);if err!=nil{http.Error(w,"db error",500);return}
 result,err:=tx.ExecContext(r.Context(),`INSERT INTO form_versions(form_id,tenant_id,version_number,schema_json,published_at) VALUES(?,?,?,?,?)`,id,tenant,n,schema,time.Now());if err!=nil{http.Error(w,"publish failed",500);return};versionID,err:=result.LastInsertId();if err!=nil{http.Error(w,"publish failed",500);return}
 _,err=tx.ExecContext(r.Context(),`UPDATE forms SET status='published',current_version_id=?,updated_at=? WHERE id=? AND tenant_id=?`,versionID,time.Now(),id,tenant);if err!=nil{http.Error(w,"publish failed",500);return};if err=tx.Commit();err!=nil{http.Error(w,"publish failed",500);return};writeJSON(w,200,map[string]any{"versionId":versionID,"versionNumber":n})
}
func (h *FormsHandler) GetPublicForm(w http.ResponseWriter,r *http.Request){
 id,err:=formID(r);if err!=nil{http.Error(w,"invalid id",400);return};var versionID int64;var number int;var schema []byte
 err=h.DB.QueryRowContext(r.Context(),`SELECT fv.id,fv.version_number,fv.schema_json FROM forms f JOIN form_versions fv ON fv.id=f.current_version_id WHERE f.id=? AND f.status='published'`,id).Scan(&versionID,&number,&schema);if errors.Is(err,sql.ErrNoRows){http.Error(w,"form not found",404);return};if err!=nil{http.Error(w,"load failed",500);return}
 w.Header().Set("Cache-Control","no-store");writeJSON(w,200,map[string]any{"formId":id,"versionId":versionID,"versionNumber":number,"schema":json.RawMessage(schema)})
}
func writeJSON(w http.ResponseWriter,status int,value any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(value)}
