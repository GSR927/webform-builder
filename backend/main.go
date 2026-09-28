package main

import (
 "database/sql"
 "log"
 "net/http"
 "os"
 "time"

 "github.com/go-chi/chi/v5"
 "github.com/go-chi/chi/v5/middleware"
 _ "github.com/go-sql-driver/mysql"
 "github.com/example/webform-builder/backend/handlers"
 "github.com/example/webform-builder/backend/queue"
 "github.com/example/webform-builder/backend/ratelimit"
)
func main(){
 db,err:=sql.Open("mysql",getenv("MYSQL_DSN","webform:webform@tcp(localhost:3306)/webform?parseTime=true&loc=UTC"));if err!=nil{log.Fatal(err)};defer db.Close();db.SetMaxOpenConns(25);db.SetMaxIdleConns(25);db.SetConnMaxLifetime(5*time.Minute)
 if err=db.Ping();err!=nil{log.Fatalf("mysql unavailable: %v",err)}
 publisher,err:=queue.NewPublisher(getenv("AMQP_URL","amqp://guest:guest@localhost:5672/"));if err!=nil{log.Fatalf("rabbitmq unavailable: %v",err)};defer publisher.Close()
 forms:=&handlers.FormsHandler{DB:db};auth:=&handlers.AuthHandler{DB:db};access:=&handlers.AccessHandler{DB:db};subs:=&handlers.SubmissionsHandler{DB:db,Publisher:publisher,Limiter:ratelimit.NewTokenBucketLimiter(20,time.Second)}
 r:=chi.NewRouter();r.Use(middleware.Logger,middleware.Recoverer,middleware.Timeout(10*time.Second))
 r.Post("/api/login",auth.Login)
 r.Get("/f/{id}",forms.GetPublicForm);r.Post("/f/{id}/submit",subs.Submit)
 r.Route("/api",func(r chi.Router){r.Use(auth.Require);r.Get("/me",auth.Me);r.With(auth.CSRF).Post("/logout",auth.Logout);r.Route("/forms",func(r chi.Router){r.With(auth.CSRF).Post("/",forms.CreateForm);r.Get("/{id}/draft",forms.GetDraft);r.With(auth.CSRF).Put("/{id}/draft",forms.UpdateDraft);r.With(auth.CSRF).Post("/{id}/publish",forms.PublishForm);r.Get("/{id}/submissions",access.List);r.Get("/{id}/submissions/export",access.Export)})})
 r.Get("/healthz",func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte("ok"))})
 addr:=getenv("ADDR",":8080");log.Printf("listening on %s",addr);log.Fatal(http.ListenAndServe(addr,r))
}
func getenv(key,fallback string)string{if v:=os.Getenv(key);v!=""{return v};return fallback}
