package main
import (
 "database/sql"
 "fmt"
 "log"
 "os"
 "strings"
 "golang.org/x/crypto/bcrypt"
 _ "github.com/go-sql-driver/mysql"
)
func main(){dsn:=os.Getenv("MYSQL_DSN");if dsn==""{log.Fatal("MYSQL_DSN required")};email:=strings.ToLower(strings.TrimSpace(os.Getenv("DEMO_EMAIL")));password:=os.Getenv("DEMO_PASSWORD");if email==""||len(password)<12{log.Fatal("set DEMO_EMAIL and DEMO_PASSWORD (at least 12 characters)")};db,err:=sql.Open("mysql",dsn);if err!=nil{log.Fatal(err)};defer db.Close();if err=db.Ping();err!=nil{log.Fatal(err)};hash,err:=bcrypt.GenerateFromPassword([]byte(password),bcrypt.DefaultCost);if err!=nil{log.Fatal(err)};var tenant int64=1
 var exists int;err=db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE id=?`,tenant).Scan(&exists);if err!=nil{log.Fatal(err)};if exists==0{log.Fatal("tenant 1 does not exist; seed tenant first")}
 _,err=db.Exec(`INSERT INTO users(tenant_id,name,email,password_hash) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE password_hash=VALUES(password_hash)`,tenant,"Demo customer",email,string(hash));if err!=nil{log.Fatal(err)};fmt.Println("demo user created for tenant 1")
}
