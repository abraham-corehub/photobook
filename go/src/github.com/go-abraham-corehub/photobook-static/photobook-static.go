package main

import (
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//Response type is to send JSON data from Server to Client
type Response struct {
	Data []string
}

//MenuItems is a custom type to store Menu items loaded dynamicaly on the Web Page's Header Bar
type MenuItems struct {
	Items string
	Flag  bool
}

//AppData is a custom type to store the Data related to the Application
type AppData struct {
	Title          string
	User           *AppUser
	MenuItemsRight []MenuItems
	Page           *PageData
	Table          *DBTable
}

//PageData is a custom type to store Title and Content / Body of the Web Page to be displayed
type PageData struct {
	Title string
	Body  string
}

//AppUser is a custom type to store the User's Name and access level (Role)
type AppUser struct {
	Name string
	Role int
	ID   int
}

//DBTable is custom
type DBTable struct {
	Header RowData
	Rows   []RowData
}

//RowData is custom
type RowData struct {
	Index int
	Row   []ColData
}

//ColData is custom
type ColData struct {
	Index int
	Value string
}

var templates *template.Template

// TemplateData type
type TemplateData struct {
	Title string
}

const dataDir = "data"
const pageDir = dataDir + "/page"
const tmplDir = "tmpl/mdl"

var pathDB = "db/pb.db"

var aD *AppData

//var templates = template.Must(template.ParseFiles(tmplDir+"/"+"head.html", tmplDir+"/"+"login.html", tmplDir+"/"+"admin.html", tmplDir+"/"+"admin.html"))

func main() {
	//testFsm()
	startWebApp()
}

func startWebApp() {
	initialize()
	mux := http.NewServeMux()
	fileServer := http.FileServer(neuteredFileSystem{http.Dir(tmplDir + "/static/")})
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("/", handlerLogin)
	mux.HandleFunc("/login", handlerAuthenticate)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

//To disable Directory Listing
//https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings
type neuteredFileSystem struct {
	fs http.FileSystem
}

//To disable Directory Listing
//https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings
func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if _, err := nfs.fs.Open(index); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func parseTemplates() {

	nUITs := []string{
		"/head",
		"/login",
		"/admin",
		"/user",
	}

	for i := 0; i < len(nUITs); i++ {
		nUITs[i] = tmplDir + nUITs[i] + ".html"
	}

	templates = template.Must(template.ParseFiles(nUITs...))
}

func initialize() {
	parseTemplates()
	aD = &AppData{}
	aD.User = &AppUser{}
	aD.Page = &PageData{}
	aD.Title = "PhotoBook"
}

func handlerLogin(w http.ResponseWriter, r *http.Request) {
	tmplStr := "login"
	renderTemplate(w, tmplStr, aD)
}

func renderTemplate(w http.ResponseWriter, tmpl string, aD *AppData) {
	err := templates.ExecuteTemplate(w, tmpl+".html", aD)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlerAuthenticate(w http.ResponseWriter, r *http.Request) {
	tmplStr := "login"
	var isValid bool
	var sessionToken string
	if r.Method == "POST" {
		c, err := r.Cookie("sessionToken")
		if err != nil {
			if err == http.ErrNoCookie {
				if err := r.ParseForm(); err != nil {
					fmt.Fprintf(w, "ParseForm() err: %v", err)
					return
				}
				uN := r.Form["username"]
				pW := r.Form["password"]

				pWH := sha1.New()
				pWH.Write([]byte(pW[0]))

				pWHS := hex.EncodeToString(pWH.Sum(nil))

				aD.User, isValid = dbCheckCredentials(uN[0], pWHS)
				if isValid {
					sessionToken, dTSExpr := setCookie(w)
					dbStoreSession(sessionToken, aD.User, dTSExpr)
					tmplStr = "admin"
					aD.Page.Title = "Administrator"
					aD.Page.Body = "This is the Admin page"
					dBT, isNotEmpty := dbGetUsers()
					if isNotEmpty {
						aD.Table = &dBT
					}
					renderTemplate(w, tmplStr, aD)
					return
				}
			} else {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		} else {
			sessionToken = c.Value
			aD.User, isValid = dbGetUserFromSession(sessionToken)
			if isValid {
				switch aD.User.Role {
				case -7:
					tmplStr = "admin"
					aD.Page.Title = "Administrator"
					aD.Page.Body = "This is the Admin page"
					dBT, isNotEmpty := dbGetUsers()
					if isNotEmpty {
						aD.Table = &dBT
					}
				default:
					tmplStr = "user"
					aD.Page.Title = aD.User.Name
					aD.Page.Body = "This is your Home page"
				}
				renderTemplate(w, tmplStr, aD)
				return
			}
			w.Write([]byte("Invalid Username"))
			return
		}
	}
}

func dbGetUserFromSession(sessionToken string) (*AppUser, bool) {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		log.Fatal(err)
		return aD.User, false
	}
	defer db.Close()

	queryString := "select id_user from session where id == \"" + sessionToken + "\""
	rows, err := db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var idUser int
		err = rows.Scan(&idUser)
		if err != nil {
			log.Fatal(err)
		}
		aD.User.ID = idUser
	} else {
		return aD.User, false
	}

	queryString = "select name, role from user where id == " + strconv.Itoa(aD.User.ID)
	rows, err = db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var name string
		var role int
		err = rows.Scan(&name, &role)
		if err != nil {
			log.Fatal(err)
			return aD.User, false
		}
		aD.User.Name = name
		aD.User.Role = role
		return aD.User, true
	}

	return aD.User, false
}

func setCookie(w http.ResponseWriter) (string, time.Time) {
	uuid, err := newUUID()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	sessionToken := uuid
	dTSExpr := time.Now().Add(120 * time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:    "sessionToken",
		Value:   sessionToken,
		Expires: dTSExpr,
	})
	return sessionToken, dTSExpr
}

func dbCheckCredentials(username string, password string) (*AppUser, bool) {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	username, fl := conditionString(username)
	if fl {
		username = "\"" + username + "\""

		password, fl = conditionString(password)

		if fl {
			password = "\"" + password + "\""

			queryString := "select name, role, id from user where username == " + username + " and password == " + password
			rows, err := db.Query(queryString)
			if err != nil {
				log.Fatal(err)
			}

			defer rows.Close()

			if rows.Next() {
				var name string
				var role int
				var id int
				err = rows.Scan(&name, &role, &id)
				if err != nil {
					log.Fatal(err)
				} else {
					aD.User.Name = name
					aD.User.Role = role
					aD.User.ID = id
					return aD.User, true
				}
			}
		}
	}

	return aD.User, false
}

func dbGetUsers() (DBTable, bool) {
	db, err := sql.Open("sqlite3", pathDB)
	dbTable := DBTable{}
	dbTable.Header = RowData{0, []ColData{{Index: 0, Value: "name"}, {Index: 1, Value: "username"}}}
	dbTable.Rows = make([]RowData, 0)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + dbTable.Header.Row[0].Value + ` from user where ` + dbTable.Header.Row[1].Value + ` != "admin"`
	rows, err := db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			log.Fatal(err)
		} else {
			dbTable.Rows = append(dbTable.Rows, RowData{Index: len(dbTable.Rows) + 1, Row: []ColData{{Value: name}}})
		}
	}
	if len(dbTable.Rows) > 0 {
		return dbTable, true
	}
	return dbTable, false
}

func dbStoreSession(sessionToken string, aU *AppUser, dTSExpr time.Time) {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		log.Fatal(err)
	}
	statement, err := db.Prepare(`PRAGMA foreign_keys = true;`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	statement, err = db.Prepare("INSERT INTO session (id, id_user, datetimestamp_lastlogin) VALUES (?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec(sessionToken, strconv.Itoa(aU.ID), strconv.FormatInt(dTSExpr.Unix(), 10))
	if err != nil {
		log.Fatal(err)
	}
}

func conditionString(str string) (string, bool) {
	flag := true
	strN := str
	charsTrim := []byte{
		' ',
		'\\',
		'"',
	}
	for _, cH := range charsTrim {
		strN = strings.ReplaceAll(strN, string(cH), "")
	}
	if len(str) != len(strN) {
		flag = false
	}
	return str, flag
}

// newUUID generates a random UUID according to RFC 4122
// https://play.golang.org/p/w7qciopoosz
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
