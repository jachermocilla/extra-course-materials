package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Book holds the information for a single book.
type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

// books is our in-memory list of all books.
var books []Book

// nextID tracks the ID to assign to the next new book.
var nextID int = 1

// mu prevents data corruption when multiple requests arrive at the same time.
var mu sync.Mutex

// sendJSON sends a JSON response back to the client.
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// readJSON reads the request body and decodes JSON into dest.
func readJSON(r *http.Request, dest interface{}) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

// getIDFromURL extracts the numeric ID from the end of a URL path.
func getIDFromURL(path string) (int, error) {
	parts := strings.Split(path, "/")
	lastPart := parts[len(parts)-1]
	return strconv.Atoi(lastPart)
}

// listBooks handles GET /books — returns all books.
func listBooks(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if books == nil {
		sendJSON(w, http.StatusOK, []Book{})
		return
	}
	sendJSON(w, http.StatusOK, books)
}

// createBook handles POST /books — adds a new book.
func createBook(w http.ResponseWriter, r *http.Request) {
	var newBook Book

	err := readJSON(r, &newBook)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Could not read request. Make sure you sent valid JSON.",
		})
		return
	}

	if newBook.Title == "" || newBook.Author == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Title and Author are required.",
		})
		return
	}

	mu.Lock()
	newBook.ID = nextID
	nextID++
	books = append(books, newBook)
	mu.Unlock()

	sendJSON(w, http.StatusCreated, newBook)
}

// getBook handles GET /books/{id} — returns one book.
func getBook(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r.URL.Path)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid ID. Please provide a number.",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for _, book := range books {
		if book.ID == id {
			sendJSON(w, http.StatusOK, book)
			return
		}
	}

	sendJSON(w, http.StatusNotFound, map[string]string{
		"error": "Book not found.",
	})
}

// updateBook handles PUT /books/{id} — replaces a book's data.
func updateBook(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r.URL.Path)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid ID. Please provide a number.",
		})
		return
	}

	var updatedBook Book
	err = readJSON(r, &updatedBook)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Could not read request. Make sure you sent valid JSON.",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i, book := range books {
		if book.ID == id {
			updatedBook.ID = id
			books[i] = updatedBook
			sendJSON(w, http.StatusOK, updatedBook)
			return
		}
	}

	sendJSON(w, http.StatusNotFound, map[string]string{
		"error": "Book not found.",
	})
}

// deleteBook handles DELETE /books/{id} — removes a book.
func deleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r.URL.Path)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid ID. Please provide a number.",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i, book := range books {
		if book.ID == id {
			books = append(books[:i], books[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	sendJSON(w, http.StatusNotFound, map[string]string{
		"error": "Book not found.",
	})
}

// booksRouter directs requests to the right handler.
func booksRouter(w http.ResponseWriter, r *http.Request) {
	isCollection := r.URL.Path == "/books" || r.URL.Path == "/books/"

	if isCollection {
		switch r.Method {
		case http.MethodGet:
			listBooks(w, r)
		case http.MethodPost:
			createBook(w, r)
		default:
			sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "Method not allowed.",
			})
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		getBook(w, r)
	case http.MethodPut:
		updateBook(w, r)
	case http.MethodDelete:
		deleteBook(w, r)
	default:
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed.",
		})
	}
}

func main() {
	http.HandleFunc("/books", booksRouter)
	http.HandleFunc("/books/", booksRouter)

	fmt.Println("Books API is running!")
	fmt.Println("Open: http://localhost:8080/books")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server failed to start:", err)
	}
}
