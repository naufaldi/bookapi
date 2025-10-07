# 📖 RFC: Personal Book Tracking API

## 1. Motivation

A REST API to help users track books they’ve read, want to read, or are currently reading. Inspired by Goodreads, but lightweight and personal.

---

## 2. Resources

- **Users**: register, login, manage their personal lists.
- **Books**: metadata available publicly (ISBN, title, description, genre, publisher).
- **Wishlists / Reading Lists / Ratings**: private to each user.

---

## 3. Endpoints

### Public

- `POST /users/register` → create new user.
- `POST /users/login` → authenticate, return JWT token.
- `GET /books` → list all books (with pagination + filters: `genre`, `publisher`).
- `GET /books/{isbn}` → get details of a single book.

### Private (requires login)

- `POST /users/{id}/wishlist` → add a book to wishlist.
- `GET /users/{id}/wishlist` → view wishlist.
- `POST /users/{id}/reading` → mark a book as currently reading.
- `GET /users/{id}/reading` → list books currently being read.
- `POST /users/{id}/finished` → mark a book as finished.
- `GET /users/{id}/finished` → list finished books.
- `POST /books/{isbn}/rating` → rate a book (1–5 stars).
- `GET /books/{isbn}/rating` → see average rating + your rating.

---

## 4. Status Codes & Rules

- `200 OK` → success GET.
- `201 Created` → new resource added (register, add to wishlist).
- `400 Bad Request` → invalid input (missing title, bad rating).
- `401 Unauthorized` → not logged in for private endpoints.
- `404 Not Found` → book or user doesn’t exist.

Validation:

- Rating must be between 1–5.
- ISBN must be valid format.
- User must exist before adding wishlist.

---

## 5. User Stories

*As a user…*

1. **Registration & Login**
    - I can create an account with email/password.
    - I can log in and receive a token to use for other endpoints.
2. **Browse Books**
    - I can see all books without logging in.
    - I can search/filter books by genre or publisher.
    - I can open a single book’s detail page.
3. **Wishlist & Reading Progress**
    - I can add books I want to read into my wishlist.
    - I can move a book from wishlist → currently reading.
    - I can mark a book as finished and see my finished list.
4. **Rating**
    - I can give a star rating (1–5) to a book I’ve finished.
    - I can see the average rating of any book, and my own rating if logged in.

    # DBML

```sql
Project BookLibrary {
  database_type: 'Postgres'
  Note: 'Schema for Book Tracking API (Users, Books, UserBooks, Ratings)'
}

Table users {
  id uuid [pk]
  username varchar [unique, not null] // public identity
  role varchar [not null] // e.g. USER | ADMIN
  email varchar(255) [unique, not null]
  password_hash text [not null]
  created_at timestamptz [not null]
  updated_at timestamptz [not null]
}

Table books {
  id uuid [pk]
  isbn varchar(20) [unique, not null]
  title varchar [not null]
  genre varchar
  publisher varchar
  description text
  created_at timestamptz [not null]
  updated_at timestamptz [not null]
}

Table ratings {
  id uuid [pk]
  user_id uuid [not null]
  book_id uuid [not null]
  star integer [not null] // 1..5
  created_at timestamptz [not null]
  updated_at timestamptz [not null]

  Indexes {
    (user_id, book_id) [unique] // one rating per user/book
    (book_id) // speed up avg rating per book
  }
}

Table user_books {
  user_id uuid [not null]
  book_id uuid [not null]
  status varchar(16) [not null] // WISHLIST | READING | FINISHED
  created_at timestamptz [not null]
  updated_at timestamptz [not null]

  Indexes {
    (user_id, book_id) [unique] // one row per user/book
    (user_id, status) // fast lookup by list type
  }
}

// Relationships
Ref: user_books.user_id > users.id
Ref: user_books.book_id > books.id
Ref: ratings.user_id > users.id
Ref: ratings.book_id > books.id
```

# SQL

```sql
CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username varchar UNIQUE NOT NULL,
  role varchar NOT NULL CHECK (role IN ('USER','ADMIN')),
  email varchar(255) UNIQUE NOT NULL,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE books (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  isbn varchar(20) UNIQUE NOT NULL,
  title varchar NOT NULL,
  genre varchar,
  publisher varchar,
  description text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ratings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  book_id uuid NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  star integer NOT NULL CHECK (star BETWEEN 1 AND 5),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, book_id)
);

CREATE INDEX idx_ratings_book_id ON ratings(book_id);

CREATE TABLE user_books (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  book_id uuid NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  status varchar(16) NOT NULL CHECK (status IN ('WISHLIST','READING','FINISHED')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, book_id)
);

-- Index for fast lookup by user + status
CREATE INDEX idx_user_books_user_status ON user_books (user_id, status);

```