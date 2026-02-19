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
