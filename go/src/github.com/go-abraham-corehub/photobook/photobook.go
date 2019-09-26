package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"

	"crypto/sha1"

	_ "github.com/mattn/go-sqlite3"
)

//Response type is to send JSON data from Server to Client
type Response struct {
	Data *AppData
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
	MenuItemsLeft  []MenuItems
	MenuItemsRight []MenuItems
	Page           *PageData
	Table          *DBTable
	State          string
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
}

//DBTable is custom
type DBTable struct {
	Header RowData
	Rows   []RowData
}

//RowData is custom
type RowData struct {
	Row []ColData
}

//ColData is custom
type ColData struct {
	Value string
}

const dataDir = "data"
const pageDir = dataDir + "/page"
const tmplDir = "tmpl/mdl"

var pathDB = "db/pb.db"

var aD *AppData

var templates = template.Must(template.ParseFiles(tmplDir+"/"+"login.html", tmplDir+"/"+"home.html"))

func main() {
	startWebApp()
}

func startWebApp() {

	http.Handle("/static/", //final url can be anything
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(tmplDir+"/"+"static"))))
	http.HandleFunc("/", handlerLogin)
	http.HandleFunc("/authenticate", handlerAuthenticate)
	http.HandleFunc("/ajax", handlerAjax)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func init() {
	aD = &AppData{}
	aD.User = &AppUser{}
	aD.Page = &PageData{}
	aD.Title = "PhotoBook"
}

func handlerLogin(w http.ResponseWriter, r *http.Request) {
	state := "login"
	aD, err := loadPage(state, "", 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, state, aD)
}

func handlerAuthenticate(w http.ResponseWriter, r *http.Request) {
	var err error
	uN := r.FormValue("username")
	pW := r.FormValue("password")

	pWH := sha1.New()
	pWH.Write([]byte(pW))

	pWHS := hex.EncodeToString(pWH.Sum(nil))

	state := "login"
	user, role, isValid := dbCheckCredentials(uN, pWHS)
	if isValid {
		state = "home"
	}
	aD, err = loadPage(state, user, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, state, aD)
}

func dbCheckCredentials(username string, password string) (string, int, bool) {
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

			queryString := "select name, role from user where username == " + username + " and password == " + password
			rows, err := db.Query(queryString)
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
				} else {
					return name, role, true
				}
			}
		}
	}

	return "", 0, false
}

func dbGetUsers() (DBTable, bool) {
	db, err := sql.Open("sqlite3", pathDB)
	dbTable := DBTable{}
	dbTable.Header = RowData{[]ColData{{"name"}, {"username"}}}
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
			dbTable.Rows = append(dbTable.Rows, RowData{[]ColData{{name}}})
		}
	}
	if len(dbTable.Rows) > 0 {
		return dbTable, true
	}
	return dbTable, false
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

func loadPage(state string, user string, role int) (*AppData, error) {
	aD.User.Name = user
	aD.User.Role = role
	aD.State = state
	var nameFilePageContent string
	switch aD.State {
	case "home":
		switch aD.User.Role {
		case -7:
			nameFilePageContent = "home-admin"
			aD.MenuItemsLeft = []MenuItems{
				{Items: "My Account"},
				{Items: "Quit"},
			}
			aD.MenuItemsRight = []MenuItems{
				{Items: "Create User"},
				{Items: "Upload Image"},
				{Items: "Create Album"},
				{Items: "Download Album"},
			}
			aD.Page.Title = "Administrator"

			dBT, isNotEmpty := dbGetUsers()
			if isNotEmpty {
				aD.Table = &dBT
			}

		default:
			nameFilePageContent = "home-user"
			aD.MenuItemsLeft = []MenuItems{
				{Items: "My Account"},
				{Items: "Quit"},
			}
			aD.MenuItemsRight = []MenuItems{
				{Items: "Upload Image"},
				{Items: "Create Album"},
				{Items: "Download Album"},
			}
			aD.Page.Title = aD.User.Name
		}
	default:
		nameFilePageContent = "login"
	}
	filename := pageDir + "/content-" + nameFilePageContent + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	aD.Page.Body = string(body)
	return aD, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, aD *AppData) {
	err := templates.ExecuteTemplate(w, tmpl+".html", aD)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AJAX Request Handler https://github.com/ET-CS/golang-response-examples/blob/master/ajax-json.go
func handlerAjax(w http.ResponseWriter, r *http.Request) {
	response := Response{Data: aD}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func testDb() {
	db, err := sql.Open("sqlite3", pathDB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("select * from user")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var un string
		var pw string
		err = rows.Scan(&un, &pw)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(un, pw)
	}
}
