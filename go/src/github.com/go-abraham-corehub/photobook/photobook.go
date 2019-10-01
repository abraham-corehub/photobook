package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"

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

// appFSM holds the State Transition Table and
// State Assignment Mappings which defines the State Machine
type appFSM struct {
	state string
	mSTD  map[cSIn]byte   // Maps current states to next states for predefined inputs
	mSTX  map[byte]byte   // Maps current states to next states without considering inputs
	mS2ID map[string]byte // Maps State Names to State IDs
	mID2S map[byte]string // Maps State IDs to State Names
}

type cSIn struct {
	cS byte
	in byte
}

const dataDir = "data"
const pageDir = dataDir + "/page"
const tmplDir = "tmpl/mdl"

var fsm appFSM

// sTT defines the State Transitions for every client request (ajax) corresponding to the current server state.
// current page at client side as remembered by the server, ajax input value sent by client, next page to be sent to client
var sTT = [][]string{
	{"login", "1", "home-admin"},
	{"login", "2", "home-user"},
	{"home-admin", "0", "login"},
	{"home-admin", "1", "home-admin-createUser"},
	{"home-admin-createUser", "0", "login"},
	{"home-admin-createUser", "1", "home-admin"},
	{"home-admin-createUser", "2", "home-admin"},
	{"home-admin", "2", "home-admin-viewUser"},
	{"home-admin-viewUser", "0", "login"},
	{"home-admin", "3", "home-admin-updateUser"},
	{"home-admin-updateUser", "0", "login"},
	{"home-admin", "4", "home-admin-resetUser"},
	{"home-admin-resetUser", "0", "login"},
	{"home-admin", "5", "home-admin-deleteUser"},
	{"home-admin-deleteUser", "0", "login"},
	{"home-user", "0", "login"},
}

var pathDB = "db/pb.db"

var aD *AppData

var templates = template.Must(template.ParseFiles(tmplDir+"/"+"login.html", tmplDir+"/"+"home.html"))

func main() {
	//testFsm()
	startWebApp()
}

func testFsm() {
	fsm = fsm.createStateTable(sTT)
	fsm.state = "login"
	fsm.state = fsm.run("1")
	fmt.Println(fsm.state)
	fsm.state = fsm.run("4")
	fmt.Println(fsm.state)
	fsm.state = fsm.run("1")
	fmt.Println(fsm.state)
}

func startWebApp() {
	mux := http.NewServeMux()
	fileServer := http.FileServer(neuteredFileSystem{http.Dir(tmplDir + "/static/")})
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("/", handlerLogin)
	mux.HandleFunc("/authenticate", handlerAuthenticate)
	mux.HandleFunc("/ajax", handlerAjax)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// AJAX Request Handler https://github.com/ET-CS/golang-response-examples/blob/master/ajax-json.go
func handlerAjax(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	taskID := r.Form["ID"]
	indexTableRow := r.Form["Data[]"]
	//fsm.state = fsm.run(taskID[0])
	//fmt.Println(fsm.state)
	dataStr := []string{}
	if len(taskID) > 0 && len(indexTableRow) > 0 {
		dataStr = []string{
			taskID[0],
			indexTableRow[0],
		}
	} else if len(taskID) > 0 {
		dataStr = []string{
			taskID[0],
		}
	} else {
		dataStr = []string{
			indexTableRow[0],
		}
	}

	jsonEncoder := json.NewEncoder(w)
	response := Response{Data: dataStr}
	w.Header().Set("Content-Type", "application/json")
	err := jsonEncoder.Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	state := "login"
	var isValid bool
	switch r.Method {
	case "POST":
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
			state = "home"
		}
		aD, err = loadPage(state, aD.User.Name, aD.User.Role)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	renderTemplate(w, state, aD)
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
	fsm.state = state
	var nameFilePageContent string
	switch fsm.state {
	case "home":
		switch aD.User.Role {
		case -7:
			nameFilePageContent = "home-admin"
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
			aD.Table = &DBTable{}
			nameFilePageContent = "home-user"
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

func (fsm appFSM) run(fsmInput string) string {
	nS, prs := fsm.mSTD[cSIn{fsm.mS2ID[fsm.state], fsm.mS2ID[fsmInput]}]
	if prs {
		fsm.state = fsm.mID2S[nS]
	} else {
		nS, prs = fsm.mSTX[fsm.mS2ID[fsm.state]]
		if prs {
			fsm.state = fsm.mID2S[nS]
		}
	}
	return fsm.state
}

func (fsm appFSM) createStateTable(sTT [][]string) appFSM {
	table := make([][]string, 0)
	col := make([]string, 0)

	for _, row := range sTT {
		col = append(col, []string{row[0], row[1], row[2]}...)
		table = append(table, row)
	}

	//fmt.Println(table)
	states := unique(col)
	//fmt.Println(states)
	fsm.mS2ID = make(map[string]byte)
	fsm.mID2S = make(map[byte]string)
	for i, state := range states {
		fsm.mS2ID[state] = byte(i)
		fsm.mID2S[byte(i)] = state
	}

	//fmt.Println(fsm.mID2S)
	//fmt.Println(fsm.mS2ID)
	fsm.mSTD = make(map[cSIn]byte)
	fsm.mSTX = make(map[byte]byte)

	for _, row := range table {
		if row[1] != "0" {
			fsm.mSTD[cSIn{fsm.mS2ID[row[0]], fsm.mS2ID[row[1]]}] = fsm.mS2ID[row[2]]
		} else {
			fsm.mSTX[fsm.mS2ID[row[0]]] = fsm.mS2ID[row[2]]
		}
	}
	//fsm.showMapTable()
	//fmt.Println(fsm.mID2S, fsm.mSTD, fsm.mSTX)
	return fsm
}

func (fsm appFSM) showMapTable() {
	for a, v := range fsm.mSTD {
		fmt.Println(a.cS, a.in, v)
	}
}

//unique https://www.golangprograms.com/remove-duplicate-values-from-slice.html
func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
