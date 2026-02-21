# A Simple RESTful API with Flask

This tutorial walks you through building a basic RESTful API using Flask. By the end, you'll have a working API that handles CRUD operations (Create, Read, Update, Delete) for a simple "books" resource.

---

## Prerequisites

- Python 3.8+
- Basic Python knowledge
- pip (Python package manager)

---

## 1. Setup

Install Flask:

```bash
pip install flask
```

Create a project folder:

```bash
mkdir flask-api && cd flask-api
touch app.py
```

---

## 2. The Complete App

Here's the full `app.py` — we'll break it down step by step below.

```python
from flask import Flask, jsonify, request, abort

app = Flask(__name__)

# In-memory "database"
books = [
    {"id": 1, "title": "The Great Gatsby", "author": "F. Scott Fitzgerald"},
    {"id": 2, "title": "1984",             "author": "George Orwell"},
]
next_id = 3


# --- Helper ---
def find_book(book_id):
    return next((b for b in books if b["id"] == book_id), None)


# --- Routes ---

# GET /books — list all books
@app.route("/books", methods=["GET"])
def get_books():
    return jsonify(books)


# GET /books/<id> — get a single book
@app.route("/books/<int:book_id>", methods=["GET"])
def get_book(book_id):
    book = find_book(book_id)
    if not book:
        abort(404, description="Book not found")
    return jsonify(book)


# POST /books — create a new book
@app.route("/books", methods=["POST"])
def create_book():
    global next_id
    data = request.get_json()
    if not data or "title" not in data or "author" not in data:
        abort(400, description="'title' and 'author' are required")
    book = {"id": next_id, "title": data["title"], "author": data["author"]}
    books.append(book)
    next_id += 1
    return jsonify(book), 201


# PUT /books/<id> — update a book
@app.route("/books/<int:book_id>", methods=["PUT"])
def update_book(book_id):
    book = find_book(book_id)
    if not book:
        abort(404, description="Book not found")
    data = request.get_json()
    book["title"]  = data.get("title",  book["title"])
    book["author"] = data.get("author", book["author"])
    return jsonify(book)


# DELETE /books/<id> — delete a book
@app.route("/books/<int:book_id>", methods=["DELETE"])
def delete_book(book_id):
    book = find_book(book_id)
    if not book:
        abort(404, description="Book not found")
    books.remove(book)
    return jsonify({"message": "Book deleted"}), 200


# --- Error Handlers ---
@app.errorhandler(400)
def bad_request(e):
    return jsonify(error=str(e)), 400

@app.errorhandler(404)
def not_found(e):
    return jsonify(error=str(e)), 404


if __name__ == "__main__":
    app.run(debug=True)
```

---

## 3. Breaking It Down

### App Initialization

```python
app = Flask(__name__)
```

Creates the Flask application instance. `__name__` tells Flask where to find resources relative to this file.

### In-Memory Data Store

```python
books = [...]
next_id = 3
```

We're using a plain Python list instead of a database to keep things simple. In a real app you'd use something like SQLite, PostgreSQL, or MongoDB.

### Route Decorator

```python
@app.route("/books", methods=["GET"])
```

The `@app.route` decorator maps a URL path and HTTP method to a Python function. Every route should specify its allowed methods explicitly.

### Returning JSON

```python
return jsonify(books)
```

`jsonify()` serializes Python dicts/lists to a JSON response and sets the correct `Content-Type: application/json` header automatically.

### Reading Request Body

```python
data = request.get_json()
```

`request.get_json()` parses the incoming JSON body. It returns `None` if the body isn't valid JSON or if `Content-Type` isn't `application/json`.

### Status Codes

REST APIs communicate outcome via HTTP status codes. The most common ones used here:

| Code | Meaning |
|------|---------|
| 200  | OK (default for successful GET/PUT/DELETE) |
| 201  | Created (use after POST) |
| 400  | Bad Request (invalid input) |
| 404  | Not Found |

---

## 4. Running the API

```bash
python app.py
```

You should see:

```
 * Running on http://127.0.0.1:5000
```

---

## 5. Testing with curl

**List all books:**
```bash
curl http://localhost:5000/books
```

**Get one book:**
```bash
curl http://localhost:5000/books/1
```

**Create a book:**
```bash
curl -X POST http://localhost:5000/books \
     -H "Content-Type: application/json" \
     -d '{"title": "Brave New World", "author": "Aldous Huxley"}'
```

**Update a book:**
```bash
curl -X PUT http://localhost:5000/books/1 \
     -H "Content-Type: application/json" \
     -d '{"title": "The Great Gatsby (Updated)"}'
```

**Delete a book:**
```bash
curl -X DELETE http://localhost:5000/books/1
```

---

## Quick Reference: REST Conventions

| Method | Path         | Action             |
|--------|--------------|--------------------|
| GET    | /books       | List all           |
| GET    | /books/1     | Get one by ID      |
| POST   | /books       | Create new         |
| PUT    | /books/1     | Replace/update one |
| DELETE | /books/1     | Delete one         |
