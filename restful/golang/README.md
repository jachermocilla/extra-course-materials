# Building a RESTful API in Go — Beginner's Guide
### Learn Go by building a simple Books API step by step.

---

## What We're Building

We're going to build a web API that lets you:

- **List** all books
- **Add** a new book
- **View** a single book
- **Update** a book
- **Delete** a book

We'll use only Go's built-in packages — nothing to install. Let's go!

---

## Prerequisites

- Go installed → https://go.dev/dl
- A terminal (Command Prompt, PowerShell, or any shell)
- Any text editor (VS Code recommended)

---

## Project Setup

Open your terminal and run:

```bash
mkdir books-api
cd books-api
go mod init books-api
touch main.go
```

> `go mod init` creates a `go.mod` file that tracks your project. Think of it like a name tag for your project.

---

## What is a REST API?

A REST API is a web server that listens for HTTP requests and responds with data (usually JSON).

| HTTP Method | What it does       | Example          |
|-------------|--------------------|------------------|
| GET         | Read data          | Get all books    |
| POST        | Create new data    | Add a book       |
| PUT         | Update data        | Edit a book      |
| DELETE      | Remove data        | Delete a book    |

---

## Step 1 — Package and Imports

Open `main.go` and start with this. Every Go file starts with a `package` name, and we list the packages we need in `import`.

```go
package main

// These are Go's built-in packages — no install needed.
import (
    "encoding/json" // converts Go data to/from JSON
    "fmt"           // for printing messages
    "net/http"      // for building the web server
    "strconv"       // converts strings to numbers
    "strings"       // for working with text
    "sync"          // helps prevent bugs when multiple requests come in at once
)
```

> **Tip:** In Go, if you import a package and don't use it, the program won't compile. Only import what you need.

---

## Step 2 — Define the Book

A `struct` is like a template that describes what a Book looks like — what fields it has and what type each field is.

```go
// Book holds the information for a single book.
// The part in backticks (`) tells Go how to name each field in JSON.
type Book struct {
    ID     int    `json:"id"`
    Title  string `json:"title"`
    Author string `json:"author"`
    Year   int    `json:"year"`
}
```

> **What is a struct?**
> Think of it like a form with labeled boxes. Each Book has an ID number, a Title, an Author name, and a Year.

When this gets converted to JSON it looks like:
```json
{
  "id": 1,
  "title": "The Go Programming Language",
  "author": "Donovan & Kernighan",
  "year": 2015
}
```

---

## Step 3 — Create the Data Store

Instead of a real database, we'll keep books in a **slice** (Go's version of a list). We use a `sync.Mutex` to make sure two requests don't edit the list at the exact same time.

```go
// books is our list of all books.
var books []Book

// nextID keeps track of the next ID number to assign.
var nextID int = 1

// mu is a lock. We lock it before changing the list,
// and unlock it when we're done. This prevents data corruption.
var mu sync.Mutex
```

> **What is a slice?**
> A slice is a flexible list. `[]Book` means "a list of Books". It starts empty and we can add or remove items.

> **What is a mutex?**
> Imagine two people trying to edit the same document at the same time — things get messy. A mutex is like a "do not disturb" sign. One request must finish before another can start editing.

---

## Step 4 — Two Helper Functions

These two functions will be used by all our handlers to read incoming data and send responses.

```go
// sendJSON sends a response back to the client as JSON.
// w      = the response writer (where we write our answer)
// status = the HTTP status code (200 = OK, 201 = Created, etc.)
// data   = whatever we want to send back (a book, a list, an error message)
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
    // Tell the client we're sending JSON
    w.Header().Set("Content-Type", "application/json")
    // Set the status code
    w.WriteHeader(status)
    // Convert data to JSON and write it to the response
    json.NewEncoder(w).Encode(data)
}

// readJSON reads JSON from the incoming request body into a variable.
// r    = the incoming request
// dest = a pointer to the variable we want to fill
func readJSON(r *http.Request, dest interface{}) error {
    return json.NewDecoder(r.Body).Decode(dest)
}
```

> **What is `interface{}`?**
> It means "any type". We use it here so these helpers can work with Books, slices, error messages — anything.

---

## Step 5 — GET /books (List All Books)

This handler is called when someone does `GET /books`. It returns all books as JSON.

```go
func listBooks(w http.ResponseWriter, r *http.Request) {
    // Lock so no one else changes the list while we read it
    mu.Lock()
    defer mu.Unlock() // "defer" means: run this when the function exits

    // If the list is empty, return an empty array [] instead of null
    if books == nil {
        sendJSON(w, http.StatusOK, []Book{})
        return
    }

    // Send back the full list with status 200 OK
    sendJSON(w, http.StatusOK, books)
}
```

> **What is `defer`?**
> `defer mu.Unlock()` means "unlock when this function finishes, no matter what". It's a safety net — you never forget to unlock.

---

## Step 6 — POST /books (Add a Book)

This handler reads a JSON body from the request, creates a new Book, and adds it to our list.

```go
func createBook(w http.ResponseWriter, r *http.Request) {
    // Create an empty Book variable to fill in
    var newBook Book

    // Try to read the JSON from the request body into newBook
    err := readJSON(r, &newBook)
    if err != nil {
        // If something went wrong, send back a 400 Bad Request error
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Could not read request. Make sure you sent valid JSON.",
        })
        return
    }

    // Make sure the required fields are not empty
    if newBook.Title == "" || newBook.Author == "" {
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Title and Author are required.",
        })
        return
    }

    // Lock the list, assign an ID, and add the new book
    mu.Lock()
    newBook.ID = nextID
    nextID++               // increment so the next book gets a different ID
    books = append(books, newBook) // append adds an item to a slice
    mu.Unlock()

    // Send back the created book with status 201 Created
    sendJSON(w, http.StatusCreated, newBook)
}
```

> **What is `append`?**
> `append(books, newBook)` adds `newBook` to the end of the `books` slice, like pushing an item onto a list.

> **What is `map[string]string`?**
> A `map` is like a dictionary. `map[string]string{"error": "some message"}` creates a simple key-value pair that turns into `{"error": "some message"}` in JSON.

---

## Step 7 — GET /books/{id} (Get One Book)

This handler finds a specific book by its ID.

```go
func getBook(w http.ResponseWriter, r *http.Request) {
    // Get the ID from the URL (e.g. /books/2 → "2")
    id, err := getIDFromURL(r.URL.Path)
    if err != nil {
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Invalid ID. Please provide a number.",
        })
        return
    }

    mu.Lock()
    defer mu.Unlock()

    // Loop through our books slice to find the matching one
    for _, book := range books {
        if book.ID == id {
            sendJSON(w, http.StatusOK, book)
            return
        }
    }

    // If we get here, we didn't find the book
    sendJSON(w, http.StatusNotFound, map[string]string{
        "error": "Book not found.",
    })
}
```

> **What is `for _, book := range books`?**
> This is Go's way of looping over a slice. `book` is each item in the list, one at a time. The `_` means we don't need the index number.

---

## Step 8 — PUT /books/{id} (Update a Book)

This handler replaces a book's data with new data from the request.

```go
func updateBook(w http.ResponseWriter, r *http.Request) {
    // Get the ID from the URL
    id, err := getIDFromURL(r.URL.Path)
    if err != nil {
        sendJSON(w, http.StatusBadRequest, map[string]string{
            "error": "Invalid ID. Please provide a number.",
        })
        return
    }

    // Read the new book data from the request body
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

    // Find the book with this ID and update it
    for i, book := range books {
        if book.ID == id {
            updatedBook.ID = id       // keep the same ID
            books[i] = updatedBook    // replace the old book with the new one
            sendJSON(w, http.StatusOK, updatedBook)
            return
        }
    }

    sendJSON(w, http.StatusNotFound, map[string]string{
        "error": "Book not found.",
    })
}
```

> **Why `books[i] = updatedBook`?**
> When we loop with `range`, `book` is a **copy**. To actually change the item in the slice, we use `books[i]` — the real position in the list.

---

## Step 9 — DELETE /books/{id} (Delete a Book)

This handler removes a book from the list by its ID.

```go
func deleteBook(w http.ResponseWriter, r *http.Request) {
    // Get the ID from the URL
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
            // Remove item at index i from the slice
            // This joins everything before i with everything after i
            books = append(books[:i], books[i+1:]...)
            w.WriteHeader(http.StatusNoContent) // 204 = success, no body
            return
        }
    }

    sendJSON(w, http.StatusNotFound, map[string]string{
        "error": "Book not found.",
    })
}
```

> **How does `append(books[:i], books[i+1:]...)`  work?**
> - `books[:i]` = everything before index i
> - `books[i+1:]` = everything after index i
> - Joining them removes the item at index i
> - The `...` "unpacks" the second slice so append can combine them

---

## Step 10 — URL Helper Function

We need a small function to extract the ID number from a URL like `/books/3`.

```go
// getIDFromURL pulls the last part of a URL path and converts it to an int.
// Example: "/books/42" returns 42
func getIDFromURL(path string) (int, error) {
    // Split "/books/42" into ["", "books", "42"]
    parts := strings.Split(path, "/")

    // The ID is the last part
    lastPart := parts[len(parts)-1]

    // Convert the string "42" to the integer 42
    id, err := strconv.Atoi(lastPart)
    return id, err
}
```

---

## Step 11 — Route Handler (The Traffic Director)

This single function receives ALL requests to `/books` and `/books/{id}` and sends them to the right handler based on the HTTP method.

```go
// booksRouter handles all requests to /books and /books/{id}
func booksRouter(w http.ResponseWriter, r *http.Request) {

    // Check if the URL is exactly "/books" or "/books/"
    isCollection := r.URL.Path == "/books" || r.URL.Path == "/books/"

    if isCollection {
        // Route collection endpoints
        switch r.Method {
        case http.MethodGet:    // GET /books
            listBooks(w, r)
        case http.MethodPost:   // POST /books
            createBook(w, r)
        default:
            sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
                "error": "Method not allowed.",
            })
        }
        return
    }

    // Route single-item endpoints (/books/1, /books/2, etc.)
    switch r.Method {
    case http.MethodGet:    // GET /books/{id}
        getBook(w, r)
    case http.MethodPut:    // PUT /books/{id}
        updateBook(w, r)
    case http.MethodDelete: // DELETE /books/{id}
        deleteBook(w, r)
    default:
        sendJSON(w, http.StatusMethodNotAllowed, map[string]string{
            "error": "Method not allowed.",
        })
    }
}
```

> **What is a `switch` statement?**
> A `switch` checks a value against several cases. It's cleaner than writing many `if/else if` blocks.

---

## Step 12 — The main() Function

Every Go program starts at `main()`. This is where we register our routes and start the server.

```go
func main() {
    // Register our router for both /books and /books/{id}
    http.HandleFunc("/books", booksRouter)
    http.HandleFunc("/books/", booksRouter)

    // Tell the user the server is ready
    fmt.Println("Books API is running!")
    fmt.Println("Open: http://localhost:8080/books")

    // Start listening on port 8080
    // This line blocks — the server keeps running until you stop it (Ctrl+C)
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        fmt.Println("Server failed to start:", err)
    }
}
```

---

## Run the Server

```bash
go run main.go
# Books API is running!
# Open: http://localhost:8080/books
```

---

## Testing with curl

Open a **second terminal** and try these commands one by one.

### Add a book
```bash
curl -X POST http://localhost:8080/books \
  -H "Content-Type: application/json" \
  -d '{"title":"The Go Programming Language","author":"Donovan & Kernighan","year":2015}'
```
```json
{"id":1,"title":"The Go Programming Language","author":"Donovan & Kernighan","year":2015}
```

### List all books
```bash
curl http://localhost:8080/books
```
```json
[{"id":1,"title":"The Go Programming Language","author":"Donovan & Kernighan","year":2015}]
```

### Get one book
```bash
curl http://localhost:8080/books/1
```

### Update a book
```bash
curl -X PUT http://localhost:8080/books/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn Go","author":"Donovan","year":2024}'
```

### Delete a book
```bash
curl -X DELETE http://localhost:8080/books/1
# No response body — 204 No Content means success
```

---

## Complete main.go

Here is the entire program in one file, ready to copy and paste:

```go
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
```

---

## API Quick Reference

| Method   | URL            | What it does       | Success Code   |
|----------|----------------|--------------------|----------------|
| GET      | `/books`       | List all books     | 200 OK         |
| POST     | `/books`       | Add a new book     | 201 Created    |
| GET      | `/books/{id}`  | Get one book       | 200 OK         |
| PUT      | `/books/{id}`  | Update a book      | 200 OK         |
| DELETE   | `/books/{id}`  | Delete a book      | 204 No Content |

---


