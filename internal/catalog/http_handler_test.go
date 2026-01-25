package catalog

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	service := NewService(mockRepo)
	handler := NewHTTPHandler(service)

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]Book{}, 0, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/catalog/search?q=test", nil)

		handler.Search(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, 0, errors.New("db error"))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/catalog/search", nil)

		handler.Search(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHTTPHandler_GetByISBN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	service := NewService(mockRepo)
	handler := NewHTTPHandler(service)

	testBook := Book{
		ISBN13: "1234567890123",
		Title:  "Test Book",
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().GetByISBN(gomock.Any(), "1234567890123").Return(testBook, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/catalog/books/1234567890123", nil)
		r.SetPathValue("isbn", "1234567890123")

		handler.GetByISBN(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.EXPECT().GetByISBN(gomock.Any(), "1234567890123").Return(Book{}, errors.New("book not found: 1234567890123"))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/catalog/books/1234567890123", nil)
		r.SetPathValue("isbn", "1234567890123")

		handler.GetByISBN(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
