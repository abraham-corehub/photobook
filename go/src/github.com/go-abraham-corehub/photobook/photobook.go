package main

import (
	"bytes"
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
	Item string
	Link string
}

//AppData is a custom type to store the Data related to the Application
type AppData struct {
	Title          string
	User           *AppUser
	MenuItemsRight []MenuItems
	Page           *PageData
	Table          *DBTable
	State          string
}

//PageData is a custom type to store Title and Content / Body of the Web Page to be displayed
type PageData struct {
	Name  string
	Title string
	Body  string
	ID    int
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

func main() {
	startWebApp()
	/*
		dTSExpr := time.Now().Add(-100 * time.Second).Unix()
		fmt.Println(isTimeExpired(dTSExpr))
	*/
}

func startWebApp() {
	parseTemplates()
	initialize()
	mux := http.NewServeMux()
	fileServer := http.FileServer(neuteredFileSystem{http.Dir(tmplDir + "/static/")})
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("/", handlerLogin)
	mux.HandleFunc("/login", handlerAuthenticate)
	mux.HandleFunc("/logout", handlerLogout)
	mux.HandleFunc("/user/view", handlerViewUser)
	mux.HandleFunc("/album/view", handlerViewAlbum)
	//mux.HandleFunc("/admin/user/edit", handlerAdminUserEdit)
	//mux.HandleFunc("/admin/user/reset", handlerAdminUserReset)
	//mux.HandleFunc("/admin/user/delete", handlerAdminUserDelete)
	//mux.HandleFunc("/user", handlerUser)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func getTimeStamp(t time.Time) string {
	tD := t.Format("20060102")
	tT := strings.Replace(t.Format("15.04.05.000"), ".", "", 3)
	dTS := tD + tT
	return dTS
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
		"head",
		"login",
		"home",
		"users",
		"albums",
		"images",
	}

	for i := 0; i < len(nUITs); i++ {
		nUITs[i] = tmplDir + "/" + nUITs[i] + ".html"
	}

	templates = template.Must(template.ParseFiles(nUITs...))
}

func initialize() {
	aD = &AppData{}
	aD.User = &AppUser{}
	aD.Page = &PageData{}
	aD.Table = &DBTable{}
	aD.Title = "PhotoBook"
}

func handlerLogin(w http.ResponseWriter, r *http.Request) {
	aD.User.Role = 0
	aD.State = "login"
	loadPage(w)
}

func handlerLogout(w http.ResponseWriter, r *http.Request) {
	dbClearCookie()
	initialize()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handlerAuthenticate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if verifyUser(w, r) {
			aD.State = "home"
			dbStoreSession(setCookie(w))
			switch aD.User.Role {
			case -7:
				aD.Page.Name = "users"
				aD.Page.Title = "Dashboard"
				dbGetUsers()
			default:
				aD.Page.Name = "albums"
				aD.Page.Title = "My Albums"
				aD.Page.ID = aD.User.ID
				dbGetAlbums()
			}
			loadPage(w)
		} else {
			w.Write([]byte("Invalid Username / Password"))
		}
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func handlerViewUser(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
		}
		aD.Page.ID, _ = strconv.Atoi(r.Form["id"][0])
		name := r.Form["name"][0]
		dbGetAlbums()
		aD.Page.Name = "albums"
		aD.Page.Title = name + "'s Albums"
		aD.State = "home"
		loadPage(w)
	} else {
		w.Write([]byte("User Not Authrorized!"))
	}
}

func handlerViewAlbum(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(r) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
		}
		idAlbum := r.Form["id"][0]
		aD.Page.Name = "images"
		dbGetImages(idAlbum)
		loadPage(w)

	} else {
		w.Write([]byte("User Not Authrorized!"))
	}
}

func renderTemplate(w http.ResponseWriter) {
	err := templates.ExecuteTemplate(w, aD.State+".html", aD)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loadMenuItems() {
	switch aD.User.Role {
	case -7:
		aD.MenuItemsRight = []MenuItems{
			{Item: "Create User", Link: "/createUser"},
			{Item: "Upload Image", Link: "/uploadImage"},
			{Item: "Create Album", Link: "/createAlbum"},
			{Item: "Download Album", Link: "/downloadAlbum"},
		}
	default:
		aD.Page.ID = aD.User.ID
		aD.MenuItemsRight = []MenuItems{
			{Item: "Upload Image", Link: "/uploadImage"},
			{Item: "Create Album", Link: "/createAlbum"},
			{Item: "Download Album", Link: "/downloadAlbum"},
		}
	}
}

func dbGetAlbums() bool {
	db, err := sql.Open("sqlite3", pathDB)
	aD.Table.Header = RowData{0, []ColData{{Index: 0, Value: "name"}, {Index: 1, Value: "id_user"}}}
	aD.Table.Rows = make([]RowData, 0)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + aD.Table.Header.Row[0].Value + ` from album where ` + aD.Table.Header.Row[1].Value + ` == ` + strconv.Itoa(aD.Page.ID)
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
			aD.Table.Rows = append(aD.Table.Rows, RowData{Index: len(aD.Table.Rows) + 1, Row: []ColData{{Value: name}}})
		}
	}
	if len(aD.Table.Rows) > 0 {
		return true
	}
	return false
}

func dbGetImages(idAlbum string) bool {
	db, err := sql.Open("sqlite3", pathDB)
	aD.Table.Header = RowData{0, []ColData{{Index: 0, Value: "name"}, {Index: 1, Value: "id_user"}, {Index: 2, Value: "id_album"}}}
	aD.Table.Rows = make([]RowData, 0)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + aD.Table.Header.Row[0].Value + ` from image where ` + aD.Table.Header.Row[1].Value + ` == ` + strconv.Itoa(aD.Page.ID) + ` and ` + aD.Table.Header.Row[2].Value + ` == ` + idAlbum
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
			aD.Table.Rows = append(aD.Table.Rows, RowData{Index: len(aD.Table.Rows) + 1, Row: []ColData{{Value: name}}})
		}
	}
	if len(aD.Table.Rows) > 0 {
		return true
	}
	return false
}

func dbClearCookie() {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		log.Fatal(err)
	}

	queryString := `DELETE FROM session where id_user == ` + strconv.Itoa(aD.User.ID)
	_, err = db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}
}

func isAuthorized(r *http.Request) bool {
	c, err := r.Cookie("sessionToken")
	if err != nil {
		if err == http.ErrNoCookie {
			return false
		}
	} else {
		if dbSetUserFromSession(c.Value) {
			return true
		}
	}
	return false
}

func verifyUser(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return false
	}
	uN := r.Form["username"]
	pW := r.Form["password"]

	pWH := sha1.New()
	pWH.Write([]byte(pW[0]))

	pWHS := hex.EncodeToString(pWH.Sum(nil))

	if dbCheckCredentials(uN[0], pWHS) {
		return true
	}
	return false
}

func loadPage(w http.ResponseWriter) {
	switch aD.State {
	case "home":
		loadMenuItems()
		aD.Page.loadPageBody()
	}
	renderTemplate(w)
}

func (PageData) loadPageBody() {
	var tpl bytes.Buffer
	err := templates.ExecuteTemplate(&tpl, aD.Page.Name+".html", aD)
	if err != nil {
		log.Fatal(err)
	}
	aD.Page.Body = tpl.String()
}

func dbSetUserFromSession(sessionToken string) bool {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer db.Close()

	queryString := "select id_user, datetimestamp_lastlogin from session where id == \"" + sessionToken + "\""
	rows, err := db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var idUser int
		var dTS string
		err = rows.Scan(&idUser, &dTS)
		if err != nil {
			log.Fatal(err)
		}
		aD.User.ID = idUser
		dTSExpr, _ := strconv.ParseInt(dTS, 10, 64)
		if isTimeExpired(dTSExpr) {
			return false
		}
	} else {
		return false
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
		}
		aD.User.Name = name
		aD.User.Role = role
		return true
	}

	return false
}

func isTimeExpired(dTSExpr int64) bool {
	dTSNow := time.Now()
	if dTSNow.Unix()-dTSExpr > 120 {
		return true
	}
	return false
}

func setCookie(w http.ResponseWriter) (string, time.Time) {
	uuid, err := newUUID()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	sessionToken := uuid
	dTSNow := time.Now()
	http.SetCookie(w, &http.Cookie{
		Name:    "sessionToken",
		Value:   sessionToken,
		Expires: dTSNow.Add(120 * time.Second),
	})
	return sessionToken, dTSNow
}

func dbCheckCredentials(username string, password string) bool {
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
					return true
				}
			}
		}
	}

	return false
}

func dbGetUsers() bool {
	db, err := sql.Open("sqlite3", pathDB)
	aD.Table.Header = RowData{0, []ColData{{Index: 0, Value: "id"}, {Index: 1, Value: "name"}, {Index: 2, Value: "username"}}}
	aD.Table.Rows = make([]RowData, 0)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + aD.Table.Header.Row[0].Value + `, ` + aD.Table.Header.Row[1].Value + ` from user where ` + aD.Table.Header.Row[2].Value + ` != "admin"`
	rows, err := db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		var id int
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		} else {
			aD.Table.Rows = append(aD.Table.Rows, RowData{Index: id, Row: []ColData{{Value: name}}})
		}
	}
	if len(aD.Table.Rows) > 0 {
		return true
	}
	return false
}

func dbStoreSession(sessionToken string, dTSExpr time.Time) {
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
	_, err = statement.Exec(sessionToken, strconv.Itoa(aD.User.ID), strconv.FormatInt(dTSExpr.Unix(), 10))
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
