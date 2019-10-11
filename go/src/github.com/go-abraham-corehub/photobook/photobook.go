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
	Name   string
	Title  string
	Body   string
	Author *PageAuthor
	Error  string
}

//AppUser is a custom type to store the User's Name and access level (Role)
type AppUser struct {
	Name     string
	Role     int
	ID       int
	Username string
	Password string
	Status   string
}

//PageAuthor is a custom type to store the User's Name and access level (Role)
type PageAuthor struct {
	Name string
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

// TemplateData type
type TemplateData struct {
	Title string
}

const dataDir = "data"
const pageDir = dataDir + "/page"
const tmplDir = "tmpl/mdl"

const pathDB = "db/pb.db"

var templates *template.Template
var aD *AppData

func main() {
	fmt.Println("Server Started!, Please access http://localhost:8080");
	startPhotoBook()
}

func startPhotoBook() {
	parseTemplates()
	initialize()
	mux := http.NewServeMux()
	fileServer := http.FileServer(neuteredFileSystem{http.Dir(tmplDir + "/static/")})
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("/", handlerRoot)
	mux.HandleFunc("/login", handlerLogin)
	mux.HandleFunc("/logout", handlerLogout)
	mux.HandleFunc("/user/view", handlerViewUser)
	mux.HandleFunc("/album/view", handlerViewAlbum)
	mux.HandleFunc("/image/view", handlerViewImage)
	//mux.HandleFunc("/admin/user/edit", handlerAdminUserEdit)
	//mux.HandleFunc("/admin/user/reset", handlerAdminUserReset)
	//mux.HandleFunc("/admin/user/delete", handlerAdminUserDelete)
	//mux.HandleFunc("/user", handlerUser)
	log.Fatal(http.ListenAndServe(":8080", mux))
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
	aD.Page.Author = &PageAuthor{}
	aD.Title = "PhotoBook"
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	clearCookie(w)
	initialize()
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return
}

func handlerLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		switch validateCredentials(w, r) {
		case 2:
			aD.State = "home"
			sessionToken, dTSExpr := setCookie(w)
			dbStoreSession(w, sessionToken, dTSExpr)
			switch aD.User.Role {
			case -7:
				aD.Page.Name = "users"
				aD.Page.Title = "Dashboard"
				dbGetUsers(w)
			default:
				aD.Page.Name = "albums"
				aD.Page.Title = "My Albums"
				aD.Page.Author.ID = aD.User.ID
				aD.Page.Author.Name = aD.User.Name
				dbGetAlbums(w)
			}
			loadPage(w)
			return
		case 1:
			//showInvalidPassword(w)
			aD.User.Status = "Non-registered Password!"
		case 0:
			//showInvalidUsername(w)
			aD.User.Status = "Non-registered Username!"
		}
		aD.User.Role = 0
		aD.State = "login"
		loadPage(w)
		return
	}
	clearCookie(w)
	initialize()
	aD.User.Role = 0
	aD.State = "login"
	loadPage(w)
}

func handlerLogout(w http.ResponseWriter, r *http.Request) {
	dbClearCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

func handlerViewUser(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(w, r) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
		}

		aD.Page.Author.ID, _ = strconv.Atoi(r.Form.Get("id"))
		aD.Page.Author.Name = r.Form.Get("name")
		dbGetAlbums(w)
		aD.Page.Name = "albums"
		aD.Page.Title = aD.Page.Author.Name + "'s Albums"
		aD.State = "home"
		loadPage(w)
	} else {
		showNotAuthorized(w)
	}
}

func handlerViewAlbum(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(w, r) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
		}
		idAlbum := r.Form.Get("id")
		nameAlbum := r.Form.Get("name")
		aD.Page.Name = "images"
		aD.Page.Title = aD.Page.Author.Name + "'s " + nameAlbum
		dbGetImages(w, idAlbum)
		aD.State = "home"
		loadPage(w)
	} else {
		showNotAuthorized(w)
	}
}

func handlerViewImage(w http.ResponseWriter, r *http.Request) {
	if isAuthorized(w, r) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
		}
		idAlbum := r.Form.Get("id")
		nameAlbum := r.Form.Get("name")
		aD.Page.Name = "images"
		aD.Page.Title = aD.Page.Author.Name + "'s " + nameAlbum
		dbGetImages(w, idAlbum)
		aD.State = "home"
		loadPage(w)
	} else {
		showNotAuthorized(w)
	}
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "sessionToken",
		Value:   "",
		Expires: time.Now(),
	})
}

func showNotAuthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<center><p>User Not Authrorized!</p><br><a href=\"/\">Login</a></center>")
}

func showInvalidUsername(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<center><p>Username <i>"+aD.User.Username+"</i> is Invalid!</p><br><a href=\"/\">Login</a></center>")
}

func showInvalidPassword(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<center><p>Password given is wrong for <i>"+aD.User.Username+"<i></p><br><a href=\"/\">Login</a></center>")
}

func showError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<center><p>Error!"+err.Error()+"</p><br><a href=\"/\">Login</a></center>")
}

func renderTemplate(w http.ResponseWriter) {
	err := templates.ExecuteTemplate(w, aD.State+".html", aD)
	if err != nil {
		showError(w, err)
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
		aD.Page.Author.ID = aD.User.ID
		aD.MenuItemsRight = []MenuItems{
			{Item: "Upload Image", Link: "/uploadImage"},
			{Item: "Create Album", Link: "/createAlbum"},
			{Item: "Download Album", Link: "/downloadAlbum"},
		}
	}
}

func isAuthorized(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie("sessionToken")
	if err != nil {
		return !(err == http.ErrNoCookie)
	}
	return dbSetUserFromSession(w, c.Value)
}

func validateCredentials(w http.ResponseWriter, r *http.Request) int {
	if err := r.ParseForm(); err != nil {
		showError(w, err)
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return 0
	}

	uN := r.Form.Get("username")
	pW := r.Form.Get("password")
	uN, isClean := conditionString(uN)
	if isClean {
		aD.User.Username = uN
		if dbIsUsernameValid(w, uN) {
			pW, isClean := conditionString(pW)
			if isClean {
				aD.User.Password = pW
				pWH := sha1.New()
				pWH.Write([]byte(pW))

				pWHS := hex.EncodeToString(pWH.Sum(nil))

				if dbCheckCredentials(w, uN, pWHS) {
					return 2
				}
				return 1
			}
		}
	}
	return 0
}

func dbIsUsernameValid(w http.ResponseWriter, username string) bool {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	defer db.Close()

	uN := "\"" + username + "\""
	queryString := "select * from user where username == " + uN
	rows, err := db.Query(queryString)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		return true
	}
	return false
}

func loadPage(w http.ResponseWriter) {
	switch aD.State {
	case "home":
		loadMenuItems()
		aD.Page.loadPageBody(w)
	}
	renderTemplate(w)
}

func (PageData) loadPageBody(w http.ResponseWriter) {
	var tpl bytes.Buffer
	err := templates.ExecuteTemplate(&tpl, aD.Page.Name+".html", aD)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	aD.Page.Body = tpl.String()
}

func dbSetUserFromSession(w http.ResponseWriter, sessionToken string) bool {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
		return false
	}
	defer db.Close()

	queryString := "select id_user, datetimestamp_lastlogin from session where id == \"" + sessionToken + "\""
	rows, err := db.Query(queryString)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var idUser int
		var dTS string
		err = rows.Scan(&idUser, &dTS)
		if err != nil {
			showError(w, err)
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
		showError(w, err)
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var name string
		var role int
		err = rows.Scan(&name, &role)
		if err != nil {
			showError(w, err)
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
		showError(w, err)
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

func dbCheckCredentials(w http.ResponseWriter, username string, password string) bool {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	defer db.Close()
	username = "\"" + username + "\""
	password = "\"" + password + "\""

	queryString := "select name, role, id from user where username == " + username + " and password == " + password
	rows, err := db.Query(queryString)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var name string
		var role int
		var id int
		err = rows.Scan(&name, &role, &id)
		if err != nil {
			showError(w, err)
			log.Fatal(err)
		} else {
			aD.User.Name = name
			aD.User.Role = role
			aD.User.ID = id
			return true
		}
	}

	return false
}

func dbGetUsers(w http.ResponseWriter) bool {
	db, err := sql.Open("sqlite3", pathDB)
	aD.Table.Header = RowData{0, []ColData{{Index: 0, Value: "id"}, {Index: 1, Value: "name"}, {Index: 2, Value: "username"}}}
	aD.Table.Rows = make([]RowData, 0)

	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + aD.Table.Header.Row[0].Value + `, ` + aD.Table.Header.Row[1].Value + ` from user where ` + aD.Table.Header.Row[2].Value + ` != "admin"`
	rows, err := db.Query(queryString)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		var id int
		err = rows.Scan(&id, &name)
		if err != nil {
			showError(w, err)
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

func dbGetAlbums(w http.ResponseWriter) bool {
	db, err := sql.Open("sqlite3", pathDB)
	aD.Table.Header = RowData{0, []ColData{{Index: 0, Value: "name"}, {Index: 1, Value: "id_user"}}}
	aD.Table.Rows = make([]RowData, 0)

	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + aD.Table.Header.Row[0].Value + ` from album where ` + aD.Table.Header.Row[1].Value + ` == ` + strconv.Itoa(aD.Page.Author.ID)
	rows, err := db.Query(queryString)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			showError(w, err)
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

func dbGetImages(w http.ResponseWriter, idAlbum string) bool {
	db, err := sql.Open("sqlite3", pathDB)
	aD.Table.Header = RowData{0, []ColData{{Index: 0, Value: "name"}, {Index: 1, Value: "id_user"}, {Index: 2, Value: "id_album"}}}
	aD.Table.Rows = make([]RowData, 0)

	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	defer db.Close()
	queryString := `select ` + aD.Table.Header.Row[0].Value + ` from image where ` + aD.Table.Header.Row[1].Value + ` == ` + strconv.Itoa(aD.Page.Author.ID) + ` and ` + aD.Table.Header.Row[2].Value + ` == ` + idAlbum
	rows, err := db.Query(queryString)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			showError(w, err)
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

func dbClearCookie(w http.ResponseWriter) {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	_, err = db.Exec("delete from session where id_user == " + strconv.Itoa(aD.User.ID))
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func dbStoreSession(w http.ResponseWriter, sessionToken string, dTSExpr time.Time) {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	statement, err := db.Prepare(`PRAGMA foreign_keys = true;`)
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}

	statement, err = db.Prepare("INSERT INTO session (id, id_user, datetimestamp_lastlogin) VALUES (?, ?, ?)")
	if err != nil {
		showError(w, err)
		log.Fatal(err)
	}
	_, err = statement.Exec(sessionToken, strconv.Itoa(aD.User.ID), strconv.FormatInt(dTSExpr.Unix(), 10))
	if err != nil {
		showError(w, err)
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
