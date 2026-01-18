package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"bookapi/internal/catalog"
	"bookapi/internal/platform/openlibrary"
)

type Config struct {
	BooksMax      int
	AuthorsMax    int
	Subjects      []string
	BatchSize     int
	FreshnessDays int
}

type OpenLibraryClient interface {
	SearchBooks(ctx context.Context, subject string, limit int) (*openlibrary.SearchResponse, error)
	GetBooksByISBN(ctx context.Context, isbns []string) (map[string]openlibrary.BookDetails, error)
	GetAuthor(ctx context.Context, authorKey string) (*openlibrary.AuthorDetails, error)
}

type Service struct {
	olClient    OpenLibraryClient
	catalogRepo catalog.Repository
	ingestRepo  Repository
	cfg         Config
}

func NewService(olClient OpenLibraryClient, catalogRepo catalog.Repository, ingestRepo Repository, cfg Config) *Service {
	return &Service{
		olClient:    olClient,
		catalogRepo: catalogRepo,
		ingestRepo:  ingestRepo,
		cfg:         cfg,
	}
}

func (s *Service) Run(ctx context.Context) (err error) {
	run := &Run{
		Status:           "RUNNING",
		ConfigBooksMax:   s.cfg.BooksMax,
		ConfigAuthorsMax: s.cfg.AuthorsMax,
		ConfigSubjects:   strings.Join(s.cfg.Subjects, ","),
		StartedAt:        time.Now(),
	}
	runID, rErr := s.ingestRepo.CreateRun(ctx, run)
	if rErr != nil {
		return rErr
	}
	run.ID = runID

	defer func() {
		now := time.Now()
		run.FinishedAt = &now
		if err != nil && run.Error == "" {
			run.Error = err.Error()
		}

		if run.Error != "" {
			run.Status = "FAILED"
		} else {
			run.Status = "COMPLETED"
		}
		if updateErr := s.ingestRepo.UpdateRun(ctx, run); updateErr != nil {
			log.Printf("Failed to update ingest run %s: %v", run.ID, updateErr)
		}
	}()

	currentBooks, err := s.catalogRepo.GetTotalBooks(ctx)
	if err != nil {
		return err
	}
	currentAuthors, err := s.catalogRepo.GetTotalAuthors(ctx)
	if err != nil {
		return err
	}

	neededBooks := s.cfg.BooksMax - currentBooks
	neededAuthors := s.cfg.AuthorsMax - currentAuthors

	if neededBooks <= 0 && neededAuthors <= 0 {
		log.Println("Ingestion targets already met. Skipping.")
		return nil
	}

	authorKeysToFetch := make(map[string]bool)
	processedISBNs := make(map[string]bool)

	for _, subject := range s.cfg.Subjects {
		if run.BooksUpserted >= neededBooks && run.AuthorsUpserted >= neededAuthors {
			break
		}

		// Discovery
		searchLimit := 100
		if neededBooks > 0 && neededBooks < 100 {
			searchLimit = neededBooks * 2
		}

		searchRes, err := s.olClient.SearchBooks(ctx, subject, searchLimit)
		if err != nil {
			run.Error = fmt.Sprintf("search failed for %s: %v", subject, err)
			return err
		}

		var isbnsToHydrate []string
		for _, doc := range searchRes.Docs {
			if len(doc.ISBN) == 0 {
				continue
			}
			isbn := doc.ISBN[0]
			// Open Library can return 10 or 13 digit ISBNs. We prefer 13.
			for _, i := range doc.ISBN {
				if len(i) == 13 {
					isbn = i
					break
				}
			}

			if processedISBNs[isbn] {
				continue
			}

			// Freshness check
			updatedAt, err := s.catalogRepo.GetBookUpdatedAt(ctx, isbn)
			if err == nil && !updatedAt.IsZero() && time.Since(updatedAt) < time.Duration(s.cfg.FreshnessDays)*24*time.Hour {
				continue
			}

			isbnsToHydrate = append(isbnsToHydrate, isbn)
			processedISBNs[isbn] = true
			if len(isbnsToHydrate) >= s.cfg.BatchSize {
				s.hydrateBatch(ctx, run, isbnsToHydrate, authorKeysToFetch)
				isbnsToHydrate = nil
				if neededBooks > 0 && run.BooksUpserted >= neededBooks {
					break
				}
			}
		}
		if len(isbnsToHydrate) > 0 {
			s.hydrateBatch(ctx, run, isbnsToHydrate, authorKeysToFetch)
		}
	}

	// Hydrate Authors
	for authorKey := range authorKeysToFetch {
		if neededAuthors > 0 && run.AuthorsUpserted >= neededAuthors {
			break
		}

		// Freshness check
		updatedAt, err := s.catalogRepo.GetAuthorUpdatedAt(ctx, authorKey)
		if err == nil && !updatedAt.IsZero() && time.Since(updatedAt) < time.Duration(s.cfg.FreshnessDays)*24*time.Hour {
			continue
		}

		authorDetails, err := s.olClient.GetAuthor(ctx, authorKey)
		if err != nil {
			log.Printf("Failed to fetch author %s: %v", authorKey, err)
			continue
		}
		run.AuthorsFetched++

		author := &catalog.Author{
			Key:       authorKey,
			Name:      authorDetails.Name,
			BirthDate: authorDetails.BirthDate,
			Bio:       formatBio(authorDetails.Bio),
		}

		rawJSON, _ := json.Marshal(authorDetails)
		if err := s.catalogRepo.UpsertAuthor(ctx, author, rawJSON); err != nil {
			log.Printf("Failed to upsert author %s: %v", authorKey, err)
			continue
		}
		run.AuthorsUpserted++
		_ = s.ingestRepo.LinkAuthorToRun(ctx, run.ID, authorKey)
	}

	return nil
}

func (s *Service) hydrateBatch(ctx context.Context, run *Run, isbns []string, authorKeys map[string]bool) {
	batch, err := s.olClient.GetBooksByISBN(ctx, isbns)
	if err != nil {
		log.Printf("Failed to hydrate batch: %v", err)
		return
	}
	run.BooksFetched += len(batch)

	for bibkey, details := range batch {
		isbn := strings.TrimPrefix(bibkey, "ISBN:")

		book := &catalog.Book{
			ISBN13:        isbn,
			Title:         details.Title,
			Subtitle:      details.Subtitle,
			Description:   details.Notes,
			CoverURL:      details.Cover.Large,
			PublishedDate: details.PublishDate,
			Publisher:     formatPublishers(details.Publishers),
			Language:      "",
			PageCount:     details.NumberOfPages,
		}

		rawJSON, _ := json.Marshal(details)
		if err := s.catalogRepo.UpsertBook(ctx, book, rawJSON); err != nil {
			log.Printf("Failed to upsert book %s: %v", isbn, err)
			continue
		}
		run.BooksUpserted++
		_ = s.ingestRepo.LinkBookToRun(ctx, run.ID, isbn)

		for _, author := range details.Authors {
			// author.URL is like "/authors/OL123A"
			if author.URL != "" {
				key := strings.TrimPrefix(author.URL, "/authors/")
				authorKeys[key] = true
			}
		}
	}
}

func formatPublishers(p []openlibrary.Publisher) string {
	if len(p) == 0 {
		return ""
	}
	names := make([]string, len(p))
	for i, pub := range p {
		names[i] = pub.Name
	}
	return strings.Join(names, ", ")
}

func formatBio(bio interface{}) string {
	if b, ok := bio.(string); ok {
		return b
	}
	if m, ok := bio.(map[string]interface{}); ok {
		if v, ok := m["value"].(string); ok {
			return v
		}
	}
	return ""
}
