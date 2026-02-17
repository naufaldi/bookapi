package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// Connect to database
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/booklibrary"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Generate seed data
	count := 10000
	log.Printf("Generating %d books...", count)

	genres := []string{"Fiction", "Science Fiction", "History", "Science", "Technology", "Romance", "Mystery", "Biography", "Philosophy", "Art"}
	languages := []string{"en", "es", "fr", "de", "it", "pt", "zh", "ja"}
	publishers := []string{"Penguin", "HarperCollins", "Oxford", "Cambridge", "MIT Press", "Springer", "Wiley", "Elsevier"}

	// Use COPY for bulk insert (much faster than individual inserts)
	var sb strings.Builder
	sb.WriteString("INSERT INTO books (id, isbn, title, subtitle, genre, publisher, description, published_date, publication_year, page_count, language, cover_url, search_vector, created_at, updated_at) VALUES ")

	now := time.Now()
	for i := 0; i < count; i++ {
		year := 1950 + rand.Intn(75)
		pages := 100 + rand.Intn(800)
		genre := genres[rand.Intn(len(genres))]
		lang := languages[rand.Intn(len(languages))]
		pub := publishers[rand.Intn(len(publishers))]

		title := fmt.Sprintf("Book Title %d - %s", i+1, getRandomWord())
		subtitle := fmt.Sprintf("A %s Story", getRandomWord())
		desc := fmt.Sprintf("This is a book about %s. It explores the fundamental concepts and provides insights into the subject matter.", getRandomWord())

		searchVector := fmt.Sprintf("'%s %s %s'", title, subtitle, desc)

		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf(
			"(gen_random_uuid(), '978-%08d', '%s', '%s', '%s', '%s', '%s', '%d-01-01', %d, %d, '%s', NULL, to_tsvector('english', %s), '%s', '%s')",
			i+1, title, subtitle, genre, pub, desc, year, year, pages, lang, searchVector, now.Format(time.RFC3339), now.Format(time.RFC3339),
		))

		if (i+1)%1000 == 0 {
			log.Printf("Generated %d/%d books", i+1, count)
		}
	}

	// Execute bulk insert
	log.Println("Inserting books into database...")
	_, err = pool.Exec(ctx, sb.String())
	if err != nil {
		log.Fatalf("Failed to insert books: %v", err)
	}

	log.Printf("Successfully inserted %d books!", count)

	// Verify count
	var total int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM books").Scan(&total)
	log.Printf("Total books in database: %d", total)
}

func getRandomWord() string {
	words := []string{
		"Adventure", "Mystery", "Journey", "Discovery", "Secrets", "Dreams", "Hope",
		"Love", "War", "Peace", "Science", "Nature", "Technology", "History", "Future",
		"Past", "Present", "Reality", "Imagination", "Wisdom", "Life", "Death",
		"Light", "Darkness", "World", "Universe", "Time", "Space", "Mind", "Soul",
	}
	return words[rand.Intn(len(words))]
}
