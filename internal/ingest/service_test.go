package ingest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bookapi/internal/catalog"
	"bookapi/internal/platform/openlibrary"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockOLClient struct {
	mock.Mock
}

func (m *mockOLClient) SearchBooks(ctx context.Context, subject string, limit int) (*openlibrary.SearchResponse, error) {
	args := m.Called(ctx, subject, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*openlibrary.SearchResponse), args.Error(1)
}

func (m *mockOLClient) GetBooksByISBN(ctx context.Context, isbns []string) (map[string]openlibrary.BookDetails, error) {
	args := m.Called(ctx, isbns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]openlibrary.BookDetails), args.Error(1)
}

func (m *mockOLClient) GetAuthor(ctx context.Context, authorKey string) (*openlibrary.AuthorDetails, error) {
	args := m.Called(ctx, authorKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*openlibrary.AuthorDetails), args.Error(1)
}

type mockCatalogRepo struct {
	mock.Mock
}

func (m *mockCatalogRepo) UpsertBook(ctx context.Context, book *catalog.Book, rawJSON []byte) error {
	args := m.Called(ctx, book, rawJSON)
	return args.Error(0)
}

func (m *mockCatalogRepo) UpsertAuthor(ctx context.Context, author *catalog.Author, rawJSON []byte) error {
	args := m.Called(ctx, author, rawJSON)
	return args.Error(0)
}

func (m *mockCatalogRepo) GetTotalBooks(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockCatalogRepo) GetTotalAuthors(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockCatalogRepo) GetBookUpdatedAt(ctx context.Context, isbn13 string) (time.Time, error) {
	args := m.Called(ctx, isbn13)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *mockCatalogRepo) GetAuthorUpdatedAt(ctx context.Context, key string) (time.Time, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Time), args.Error(1)
}

type mockIngestRepo struct {
	mock.Mock
}

func (m *mockIngestRepo) CreateRun(ctx context.Context, run *Run) (string, error) {
	args := m.Called(ctx, run)
	return args.String(0), args.Error(1)
}

func (m *mockIngestRepo) UpdateRun(ctx context.Context, run *Run) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}

func (m *mockIngestRepo) LinkBookToRun(ctx context.Context, runID string, isbn13 string) error {
	args := m.Called(ctx, runID, isbn13)
	return args.Error(0)
}

func (m *mockIngestRepo) LinkAuthorToRun(ctx context.Context, runID string, authorKey string) error {
	args := m.Called(ctx, runID, authorKey)
	return args.Error(0)
}

func TestService_Run(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		BooksMax:      10,
		AuthorsMax:    5,
		Subjects:      []string{"test"},
		BatchSize:     5,
		FreshnessDays: 7,
	}

	t.Run("incremental target reached", func(t *testing.T) {
		mOL := new(mockOLClient)
		mCatalog := new(mockCatalogRepo)
		mIngest := new(mockIngestRepo)

		s := NewService(mOL, mCatalog, mIngest, cfg)

		mIngest.On("CreateRun", ctx, mock.Anything).Return("run-0", nil)
		mIngest.On("UpdateRun", ctx, mock.MatchedBy(func(run *Run) bool {
			return run.Status == "COMPLETED"
		})).Return(nil)

		mCatalog.On("GetTotalBooks", ctx).Return(10, nil)
		mCatalog.On("GetTotalAuthors", ctx).Return(5, nil)

		err := s.Run(ctx)
		assert.NoError(t, err)
		mOL.AssertNotCalled(t, "SearchBooks", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("fetches missing books and authors", func(t *testing.T) {
		mOL := new(mockOLClient)
		mCatalog := new(mockCatalogRepo)
		mIngest := new(mockIngestRepo)

		s := NewService(mOL, mCatalog, mIngest, cfg)

		mIngest.On("CreateRun", ctx, mock.Anything).Return("run-1", nil)
		mIngest.On("UpdateRun", ctx, mock.MatchedBy(func(run *Run) bool {
			return run.Status == "COMPLETED"
		})).Return(nil)

		mCatalog.On("GetTotalBooks", ctx).Return(8, nil)   // Need 2
		mCatalog.On("GetTotalAuthors", ctx).Return(4, nil) // Need 1

		searchRes := &openlibrary.SearchResponse{
			Docs: []struct {
				Key              string   `json:"key"`
				Title            string   `json:"title"`
				AuthorNames      []string `json:"author_name"`
				AuthorKeys       []string `json:"author_key"`
				ISBN             []string `json:"isbn"`
				FirstPublishYear int      `json:"first_publish_year"`
				Language         []string `json:"language"`
			}{
				{ISBN: []string{"isbn1"}, AuthorKeys: []string{"auth1"}},
				{ISBN: []string{"isbn2"}, AuthorKeys: []string{"auth2"}},
			},
		}
		mOL.On("SearchBooks", ctx, "test", 4).Return(searchRes, nil)

		mCatalog.On("GetBookUpdatedAt", ctx, "isbn1").Return(time.Time{}, nil)
		mCatalog.On("GetBookUpdatedAt", ctx, "isbn2").Return(time.Time{}, nil)

		mOL.On("GetBooksByISBN", ctx, []string{"isbn1", "isbn2"}).Return(map[string]openlibrary.BookDetails{
			"ISBN:isbn1": {Title: "Book 1", Authors: []struct {
				URL  string `json:"url"`
				Name string `json:"name"`
			}{{URL: "/authors/auth1", Name: "Author 1"}}},
			"ISBN:isbn2": {Title: "Book 2", Authors: []struct {
				URL  string `json:"url"`
				Name string `json:"name"`
			}{{URL: "/authors/auth2", Name: "Author 2"}}},
		}, nil)

		mCatalog.On("UpsertBook", ctx, mock.Anything, mock.Anything).Return(nil).Twice()
		mIngest.On("LinkBookToRun", ctx, "run-1", mock.Anything).Return(nil).Twice()

		mCatalog.On("GetAuthorUpdatedAt", ctx, "auth1").Return(time.Time{}, nil)
		mOL.On("GetAuthor", ctx, "auth1").Return(&openlibrary.AuthorDetails{Name: "Author 1"}, nil)
		mCatalog.On("UpsertAuthor", ctx, mock.Anything, mock.Anything).Return(nil)
		mIngest.On("LinkAuthorToRun", ctx, "run-1", "auth1").Return(nil)

		err := s.Run(ctx)
		assert.NoError(t, err)

		mOL.AssertExpectations(t)
		mCatalog.AssertExpectations(t)
		mIngest.AssertExpectations(t)
	})

	t.Run("skips recently updated books", func(t *testing.T) {
		mOL := new(mockOLClient)
		mCatalog := new(mockCatalogRepo)
		mIngest := new(mockIngestRepo)

		s := NewService(mOL, mCatalog, mIngest, cfg)

		mIngest.On("CreateRun", ctx, mock.Anything).Return("run-2", nil)
		mIngest.On("UpdateRun", ctx, mock.MatchedBy(func(run *Run) bool {
			return run.Status == "COMPLETED"
		})).Return(nil)

		mCatalog.On("GetTotalBooks", ctx).Return(9, nil)
		mCatalog.On("GetTotalAuthors", ctx).Return(5, nil)

		searchRes := &openlibrary.SearchResponse{
			Docs: []struct {
				Key              string   `json:"key"`
				Title            string   `json:"title"`
				AuthorNames      []string `json:"author_name"`
				AuthorKeys       []string `json:"author_key"`
				ISBN             []string `json:"isbn"`
				FirstPublishYear int      `json:"first_publish_year"`
				Language         []string `json:"language"`
			}{
				{ISBN: []string{"isbn_recent"}},
			},
		}
		mOL.On("SearchBooks", ctx, "test", 2).Return(searchRes, nil)

		mCatalog.On("GetBookUpdatedAt", ctx, "isbn_recent").Return(time.Now(), nil) // Recently updated

		err := s.Run(ctx)
		assert.NoError(t, err)

		mOL.AssertNotCalled(t, "GetBooksByISBN", mock.Anything, mock.Anything)
	})

	t.Run("deduplicates ISBNs within a run", func(t *testing.T) {
		mOL := new(mockOLClient)
		mCatalog := new(mockCatalogRepo)
		mIngest := new(mockIngestRepo)

		s := NewService(mOL, mCatalog, mIngest, cfg)

		mIngest.On("CreateRun", ctx, mock.Anything).Return("run-3", nil)
		mIngest.On("UpdateRun", ctx, mock.MatchedBy(func(run *Run) bool {
			return run.Status == "COMPLETED"
		})).Return(nil)

		mCatalog.On("GetTotalBooks", ctx).Return(8, nil)
		mCatalog.On("GetTotalAuthors", ctx).Return(5, nil)

		searchRes := &openlibrary.SearchResponse{
			Docs: []struct {
				Key              string   `json:"key"`
				Title            string   `json:"title"`
				AuthorNames      []string `json:"author_name"`
				AuthorKeys       []string `json:"author_key"`
				ISBN             []string `json:"isbn"`
				FirstPublishYear int      `json:"first_publish_year"`
				Language         []string `json:"language"`
			}{
				{ISBN: []string{"isbn_dup"}},
				{ISBN: []string{"isbn_dup"}}, // Duplicate ISBN in search results
			},
		}
		mOL.On("SearchBooks", ctx, "test", 4).Return(searchRes, nil)

		mCatalog.On("GetBookUpdatedAt", ctx, "isbn_dup").Return(time.Time{}, nil)

		mOL.On("GetBooksByISBN", ctx, []string{"isbn_dup"}).Return(map[string]openlibrary.BookDetails{
			"ISBN:isbn_dup": {Title: "Dup Book"},
		}, nil)

		mCatalog.On("UpsertBook", ctx, mock.Anything, mock.Anything).Return(nil)
		mIngest.On("LinkBookToRun", ctx, "run-3", "isbn_dup").Return(nil)

		err := s.Run(ctx)
		assert.NoError(t, err)

		mOL.AssertNumberOfCalls(t, "GetBooksByISBN", 1)
	})

	t.Run("records failure if SearchBooks fails", func(t *testing.T) {
		mOL := new(mockOLClient)
		mCatalog := new(mockCatalogRepo)
		mIngest := new(mockIngestRepo)

		s := NewService(mOL, mCatalog, mIngest, cfg)

		mIngest.On("CreateRun", ctx, mock.Anything).Return("run-4", nil)
		mIngest.On("UpdateRun", ctx, mock.MatchedBy(func(run *Run) bool {
			return run.Status == "FAILED" && run.Error != ""
		})).Return(nil)

		mCatalog.On("GetTotalBooks", ctx).Return(8, nil)
		mCatalog.On("GetTotalAuthors", ctx).Return(5, nil)

		mOL.On("SearchBooks", ctx, "test", 4).Return(nil, fmt.Errorf("search error"))

		err := s.Run(ctx)
		assert.Error(t, err)
		mIngest.AssertExpectations(t)
	})

	t.Run("records failure if GetTotalBooks fails", func(t *testing.T) {
		mOL := new(mockOLClient)
		mCatalog := new(mockCatalogRepo)
		mIngest := new(mockIngestRepo)

		s := NewService(mOL, mCatalog, mIngest, cfg)

		mIngest.On("CreateRun", ctx, mock.Anything).Return("run-5", nil)
		mIngest.On("UpdateRun", ctx, mock.MatchedBy(func(run *Run) bool {
			return run.Status == "FAILED" && run.Error != ""
		})).Return(nil)

		mCatalog.On("GetTotalBooks", ctx).Return(0, fmt.Errorf("db error"))

		err := s.Run(ctx)
		assert.Error(t, err)
		mIngest.AssertExpectations(t)
	})
}
