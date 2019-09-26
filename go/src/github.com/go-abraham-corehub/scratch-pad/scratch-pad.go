package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	testDb()
	hashTest("admin")
	hashTest("abey")
}
func hashTest(str string) []byte {
	pW := str
	pWH := sha1.New()
	pWH.Write([]byte(pW))

	pWHS := hex.EncodeToString(pWH.Sum(nil))

	fmt.Println(pW, pWHS)
	return []byte(pWHS)
}

type parseState struct {
	inTag, isPre, isScript, isStyle, isSpace, isSkip bool
}

func setParseState(b byte, pS *parseState) {
	pS.inTag = b>>5&1 == 1
	pS.isPre = b>>4&1 == 1
	pS.isScript = b>>3&1 == 1
	pS.isStyle = b>>2&1 == 1
	pS.isSpace = b>>1&1 == 1
	pS.isSkip = b&1 == 1
}

func testDb() {
	db, err := sql.Open("sqlite3", "../photobook/db/pb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("select username, password from user")
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
