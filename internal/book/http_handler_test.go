package book

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	service := NewService(mockRepo)
	handler := NewHTTPHandler(service)

	testBook := Book{
		ID:    "1",
		ISBN:  "123",
		Title: "Test",
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]Book{testBook}, 1, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/books", nil)

		handler.List(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, 0, context.DeadlineExceeded)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/books", nil)

		handler.List(w, r)

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
		ID:    "1",
		ISBN:  "123",
		Title: "Test",
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().GetByISBN(gomock.Any(), "123").Return(testBook, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/books/123", nil)
		r.SetPathValue("isbn", "123")

		handler.GetByISBN(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.EXPECT().GetByISBN(gomock.Any(), "123").Return(Book{}, ErrNotFound)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/books/123", nil)
		r.SetPathValue("isbn", "123")

		handler.GetByISBN(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
