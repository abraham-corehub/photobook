package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"

	"crypto/sha1"

	_ "github.com/mattn/go-sqlite3"
)

//Response type for JSON
type Response struct {
	Data AppData
}

//MenuItems is a custom type
type MenuItems struct {
	Items string
	Flag  bool
}

//AppData stores the Datas for the App
type AppData struct {
	Title          string
	User           *AppUser
	MenuItemsLeft  []MenuItems
	MenuItemsRight []MenuItems
	Page           *PageData
	State          string
}

//PageData is a custom type
type PageData struct {
	Title string
	Body  string
}

//AppUser is the App User
type AppUser struct {
	Name string
	Role int
}

const dataDir = "data"
const tmplDir = "tmpl/mdl"

var db *sql.DB

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
	//http.HandleFunc("/ajax", handlerAjax)
	log.Fatal(http.ListenAndServe(":8080", nil))
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
	aD := &AppData{Title: "PhotoBook"}
	var err error
	uN := r.FormValue("username")
	pW := r.FormValue("password")

	pWH := sha1.New()
	pWH.Write([]byte(pW))

	pWHS := hex.EncodeToString(pWH.Sum(nil))

	state := "login"
	role, user, isValid := dbCheckCredentials(uN, pWHS)
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

func dbCheckCredentials(username string, password string) (int, string, bool) {
	db, err := sql.Open("sqlite3", "./db/pb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	username, fl := conditionString(username)
	if !fl {
		return 0, "", false
	}

	username = "\"" + username + "\""

	password, fl = conditionString(password)
	if !fl {
		return 0, "", false
	}
	password = "\"" + password + "\""

	queryString := "select role, name from user where username == " + username + " and password == " + password
	rows, err := db.Query(queryString)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	if rows.Next() {
		var role int
		var name string
		err = rows.Scan(&role, &name)
		if err != nil {
			log.Fatal(err)
		} else {
			return role, name, true
		}
	}
	return 0, "", false
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
	aD := &AppData{Title: "PhotoBook", User: &AppUser{user, role}, State: state, Page: &PageData{}}
	var nameFilePageContent string
	switch aD.State {
	case "home":
		switch aD.User.Role {
		case -7:
			nameFilePageContent = "home-admin"
			aD.MenuItemsLeft = []MenuItems{
				{Items: "Task 1", Flag: false},
				{Items: "Task 2", Flag: true},
				{Items: "Task 3", Flag: true},
				{Items: "Task 4", Flag: true},
			}
			aD.MenuItemsRight = []MenuItems{
				{Items: "Task 1", Flag: false},
				{Items: "Task 2", Flag: true},
				{Items: "Task 3", Flag: true},
				{Items: "Task 4", Flag: true},
			}
			aD.Page.Title = "Administrator"
			aD.Page.Body = "This is the Admin Page"
		default:
			nameFilePageContent = "home-user"
			aD.MenuItemsLeft = []MenuItems{
				{Items: "Task 1", Flag: false},
				{Items: "Task 2", Flag: true},
			}
			aD.MenuItemsRight = []MenuItems{
				{Items: "Task 1", Flag: false},
				{Items: "Task 2", Flag: true},
				{Items: "Task 3", Flag: true},
			}
			aD.Page.Title = aD.User.Name
			aD.Page.Body = "This is your Home Page"
		}
	default:
		nameFilePageContent = "login"
	}
	filename := dataDir + "/" + nameFilePageContent + ".txt"
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
	menuItems := Response{}
	js, err := json.Marshal(menuItems)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
