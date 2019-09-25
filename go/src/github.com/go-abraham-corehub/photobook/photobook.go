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

//Page is a type to implement HTML Page
type Page struct {
	Title string
	Body  []byte
	User  string
}

var cUser int

//Response type for JSON
type Response struct {
	User           string
	MenuItemsLeft  []string
	MenuItemsRight []string
}

//Todo is a custom type
type Todo struct {
	Title string
	Done  bool
}

//TodoPageData is a custom type
type TodoPageData struct {
	Title     string
	User      string
	PageTitle string
	Body      string
	Todos     []Todo
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
	http.HandleFunc("/ajax", handlerAjax)
	log.Fatal(http.ListenAndServe(":8080", nil))
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

func handlerLogin(w http.ResponseWriter, r *http.Request) {
	title := "login"
	p, err := loadPage(title, "", -1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, title, p)
}

// AJAX Request Handler
// https://github.com/ET-CS/golang-response-examples/blob/master/ajax-json.go
func handlerAjax(w http.ResponseWriter, r *http.Request) {
	menuItemsLeft := []string{"Left Menu Item 01", "Left Menu Item 02", "Left Menu Item 03"}
	menuItemsRight := []string{"Right Menu Item 01", "Right Menu Item 02", "Right Menu Item 03"}

	menuItems := Response{"user", menuItemsLeft, menuItemsRight}

	js, err := json.Marshal(menuItems)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func handlerAuthenticate(w http.ResponseWriter, r *http.Request) {

	var p *TodoPageData
	p = &TodoPageData{PageTitle: "PhotoBook"}
	uN := r.FormValue("username")
	pW := r.FormValue("password")

	pWH := sha1.New()
	pWH.Write([]byte(pW))

	pWHS := hex.EncodeToString(pWH.Sum(nil))

	title := "login"
	role, user, isValid := dbCheckCredentials(uN, pWHS)
	if isValid {
		title = "home"
		switch role {
		case -7:
			p = &TodoPageData{
				PageTitle: user,
				Body:      "This is the Admin Page",
				Todos: []Todo{
					{Title: "Task 1", Done: false},
					{Title: "Task 2", Done: true},
					{Title: "Task 3", Done: true},
					{Title: "Task 4", Done: true},
				},
			}
		default:
			p = &TodoPageData{
				PageTitle: user,
				Body:      "This is your Home Page",
				Todos: []Todo{
					{Title: "Task 1", Done: false},
					{Title: "Task 2", Done: true},
				},
			}
		}

	}

	renderTemplateNew(w, title, p)

}

func renderTemplateNew(w http.ResponseWriter, tmpl string, p *TodoPageData) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loadPage(title string, user string, role int) (*Page, error) {
	p := &Page{Title: title}
	switch title {
	case "home":
		switch role {
		case -7:
			p.User = "Administrator"
			title = "home-admin"
			cUser = -7
		default:
			p.User = user
			title = "home-user"
			cUser = 1
		}
	}
	filename := dataDir + "/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	p.Body = body
	return p, nil
}
