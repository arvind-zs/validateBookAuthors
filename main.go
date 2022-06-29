package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Book struct {
	Id            int    `json:"id"`
	Title         string `json:"title"`
	Author        Author `json:"author"`
	Publication   string `json:"publication"`
	PublishedDate string `json:"published_date"`
}

type Author struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Dob       string `json:"dob"`
	PenName   string `json:"pen_name"`
}

func databaseConnection() (db *sql.DB) {
	databaseDriver := "mysql"
	databaseUser := "root"
	databasePass := "Dpyadav@123"
	databaseName := "Test"
	db, err := sql.Open(databaseDriver, databaseUser+":"+databasePass+"@tcp(127.0.0.1:3306)/"+databaseName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func getBook(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	defer db.Close()
	title := request.URL.Query().Get("title")
	includeAuthor := request.URL.Query().Get("includeAuthor")
	var rows *sql.Rows
	var err error
	if title == "" {
		rows, err = db.Query("select * from Books;")
	} else {
		rows, err = db.Query("select * from Books where title=?;", title)
	}
	if err != nil {
		log.Print(err)
	}
	books := []Book{}
	for rows.Next() {
		book := Book{}
		err = rows.Scan(&book.Id, &book.Title, &book.Author.Id, &book.Publication, &book.PublishedDate)
		if err != nil {
			log.Print(err)
		}
		if includeAuthor == "true" {
			row := db.QueryRow("select * from Authors where id=?", book.Author.Id)
			row.Scan(&book.Author.Id, &book.Author.FirstName, &book.Author.LastName, &book.Author.Dob, &book.Author.PenName)
		}
		books = append(books, book)
	}
	json.NewEncoder(response).Encode(books)

}

func getBookById(response http.ResponseWriter, request *http.Request) {

	id, err := strconv.Atoi(mux.Vars(request)["id"])

	if err != nil {
		log.Print(err)
		response.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(response).Encode(Book{})
		return
	}
	db := databaseConnection()
	defer db.Close()
	bookrow := db.QueryRow("select * from Books where id=?;", id)
	var book Book
	err = bookrow.Scan(&book.Id, &book.Title, &book.Author.Id, &book.Publication, &book.PublishedDate)
	if err != nil {
		log.Print(err)
		if err == sql.ErrNoRows {
			response.WriteHeader(404)
			json.NewEncoder(response).Encode(book)
			return
		}
	}
	authorrow := db.QueryRow("select * from Authors where id=?;", book.Author.Id)
	err = authorrow.Scan(&book.Author.Id, &book.Author.FirstName, &book.Author.LastName, &book.Author.Dob, &book.Author.PenName)
	if err != nil {
		log.Print(err)
	}
	json.NewEncoder(response).Encode(book)
	/*
		data, err := json.Marshal(book)
		if err != nil {
			response.WriteHeader(http.StatusBadRequest)
			//response.Write([]byte(`error while unmarshalling`))
		}
		response.Write(data)

	*/

}

func postBook(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	defer db.Close()
	decoder := json.NewDecoder(request.Body)
	b := Book{}
	err := decoder.Decode(&b)
	if b.Title == "" {
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Book{})
		return
	}
	var BookId int
	err = db.QueryRow("select id from Books where title=? and author_id=?;", b.Title, b.Author.Id).Scan(&BookId)
	if err == nil {
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Book{})
		return
	}

	authorRow := db.QueryRow("select id from Authors where id=?;", b.Author.Id)
	var authorId int
	err = authorRow.Scan(&authorId)
	if err != nil {
		log.Print(err)
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Book{})
		return
	}
	if !(b.Publication == "Scholastic" || b.Publication == "Pengiun" || b.Publication == "Arihanth") {
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Book{})
		return
	}
	publicationYear, err := strconv.Atoi(strings.Split(b.PublishedDate, "/")[2])
	if err != nil {
		log.Print("invalid date")
		json.NewEncoder(response).Encode(Book{})
		return
	}
	if !(publicationYear >= 1880 && publicationYear <= time.Now().Year()) {
		log.Print("invalid date")
		json.NewEncoder(response).Encode(Book{})
		return
	}
	res, err := db.Exec("INSERT INTO Books (title,author_id, publication, publishdate)\nVALUES (?,?,?,?);", b.Title, b.Author.Id, b.Publication, b.PublishedDate)
	id, _ := res.LastInsertId()
	if err != nil {
		log.Print(err)
		json.NewEncoder(response).Encode(Book{})
	} else {
		b.Id = int(id)
		json.NewEncoder(response).Encode(b)
	}
}

func postAuthor(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	defer db.Close()
	decoder := json.NewDecoder(request.Body)
	a := Author{}
	err := decoder.Decode(&a)
	fmt.Println(a)
	if a.FirstName == "" || a.Dob == "" {
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Author{})
		return
	}
	existingAuthorId := 0
	err = db.QueryRow("SELECT id from Authors where fname=? and lname=? and dob=? and penname=?", a.FirstName, a.LastName, a.Dob, a.PenName).Scan(&existingAuthorId)
	if err == nil {
		log.Print("author already exists")
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Author{})
		return
	}
	res, err := db.Exec("INSERT INTO Authors (fname, lname, dob, penname)\nVALUES (?,?,?,?);", a.FirstName, a.LastName, a.Dob, a.PenName)
	id, err := res.LastInsertId()
	if err != nil {
		log.Print(err)
		response.WriteHeader(400)
		json.NewEncoder(response).Encode(Author{})
	} else {
		a.Id = int(id)
		json.NewEncoder(response).Encode(a)
	}
}

func putAuthor(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	var author Author
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Print(err)
		return
	}
	err = json.Unmarshal(body, &author)
	if err != nil {
		log.Print(err)
		return
	}
	if author.FirstName == "" || author.LastName == "" || author.PenName == "" || author.Dob == "" {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	params := mux.Vars(request)
	ID, err := strconv.Atoi(params["id"])
	if ID <= 0 {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := db.Query("SELECT id FROM Authors WHERE id = ?", ID)
	if err != nil {
		log.Print(err)
	}
	if !res.Next() {
		log.Print("id not present")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	var id int
	err = res.Scan(&id)
	if err != nil {
		log.Print(err)
		return
	}

	_, err = db.Exec("UPDATE Authors SET first_name = ? ,last_name = ? ,dob = ? ,pen_name = ?  WHERE id =?", author.FirstName, author.LastName, author.Dob, author.PenName, ID)
	if err != nil {
		log.Print(err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func putBook(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	var book Book
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &book)
	if err != nil {
		return
	}
	if book.Title == "" {
		response.WriteHeader(http.StatusBadRequest)
		return
	}
	if !(book.Publication == "Penguin" || book.Publication == "Scholastic" || book.Publication == "Arihanth") {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	publicationDate := strings.Split(book.PublishedDate, "/")
	if len(publicationDate) < 3 {
		return
	}
	yr, _ := strconv.Atoi(publicationDate[2])
	if yr > time.Now().Year() || yr < 1880 {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	params := mux.Vars(request)
	ID, err := strconv.Atoi(params["id"])
	if ID <= 0 {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := db.Query("SELECT id FROM Authors WHERE id = ?", book.Author.Id)
	if err != nil {
		log.Print(err)
	}

	if !result.Next() {
		log.Print("author not present", book.Author.Id)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err = db.Query("SELECT * FROM Books WHERE id = ?", book.Id)
	if err != nil {
		log.Print(err)
	}
	if !result.Next() {
		log.Print("Book not present")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err = db.Query("UPDATE Books SET title = ? ,publication = ? ,published_date = ?,author_id=?  WHERE id =?", book.Title, book.Publication, book.PublishedDate, book.Author.Id, ID)
	if err != nil {
		log.Print(err)
	}
}

func deleteAuthor(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	defer db.Close()
	id, err := strconv.Atoi(mux.Vars(request)["id"])

	fmt.Println(id)
	_, err = db.Exec("delete from Books where author_id=?;", id)
	if err != nil {
		log.Print(err)
		response.WriteHeader(400)
		return
	}
	_, err = db.Exec("delete from Authors where id=?;", id)
	if err != nil {
		response.WriteHeader(400)
		return
	}
	response.WriteHeader(200)
}

func deleteBook(response http.ResponseWriter, request *http.Request) {
	db := databaseConnection()
	defer db.Close()
	id, err := strconv.Atoi(mux.Vars(request)["id"])

	fmt.Println(id)
	bookId := 0
	err = db.QueryRow("select id from Books where id=?;", id).Scan(&bookId)
	if err == nil {
		_, err = db.Exec("delete from Books where author_id=?;", id)
		if err != nil {
			response.WriteHeader(400)
			return
		}
	} else {
		response.WriteHeader(400)
		return
	}

	response.WriteHeader(200)
}

func main() {

	rout := mux.NewRouter()

	rout.HandleFunc("/book", getBook).Methods(http.MethodGet)

	rout.HandleFunc("/book/{id}", getBookById).Methods(http.MethodGet)

	rout.HandleFunc("/book", postBook).Methods(http.MethodPost)

	rout.HandleFunc("/author", postAuthor).Methods(http.MethodPost)

	rout.HandleFunc("/book/{id}", putBook).Methods(http.MethodPut)

	rout.HandleFunc("/author/{id}", putAuthor).Methods(http.MethodPut)

	rout.HandleFunc("/book/{id}", deleteBook).Methods(http.MethodDelete)

	rout.HandleFunc("/author/{id}", deleteAuthor).Methods(http.MethodDelete)

	server := http.Server{
		Addr:    ":8000",
		Handler: rout,
	}

	fmt.Println("Server started at 8000")
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
