package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bookapi/internal/auth"
	"bookapi/internal/entity"
	"bookapi/internal/store/mocks"
	"bookapi/internal/usecase"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var TestUser = entity.User{
	ID:        "test-user-id-123",
	Username:  "testuser",
	Email:     "test@example.com",
	Password:  "hashedpassword",
	Role:      "USER",
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
}

func TestUserHandler_RegisterUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockUserRepository(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	handler := NewUserHandler(mockRepo, mockSessionRepo, "test-secret")

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "success - valid registration",
			body: map[string]string{
				"email":    "new@example.com",
				"username": "newuser",
				"password": "Password123!",
			},
			setupMock: func() {
				mockRepo.EXPECT().
					GetByEmail(gomock.Any(), "new@example.com").
					Return(entity.User{}, usecase.ErrNotFound)
				mockRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "bad request - invalid JSON",
			body:           "invalid json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - missing email",
			body: map[string]string{
				"username": "newuser",
				"password": "Password123!",
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - password too weak",
			body: map[string]string{
				"email":    "new@example.com",
				"username": "newuser",
				"password": "123",
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "conflict - email already exists",
			body: map[string]string{
				"email":    "existing@example.com",
				"username": "newuser",
				"password": "Password123!",
			},
			setupMock: func() {
				mockRepo.EXPECT().
					GetByEmail(gomock.Any(), "existing@example.com").
					Return(TestUser, nil)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users/register", nil)
			if bodyMap, ok := tt.body.(map[string]string); ok {
				body, _ := json.Marshal(bodyMap)
				r = httptest.NewRequest(http.MethodPost, "/users/register", bytes.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
			}

			handler.RegisterUser(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUserHandler_LoginUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockUserRepository(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	handler := NewUserHandler(mockRepo, mockSessionRepo, "test-secret")

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "success - valid credentials",
			body: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			},
			setupMock: func() {
				hashedPassword, _ := auth.HashPassword("password123")
				user := TestUser
				user.Password = hashedPassword
				mockRepo.EXPECT().
					GetByEmail(gomock.Any(), "test@example.com").
					Return(user, nil)
				mockSessionRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "bad request - invalid JSON",
			body:           "invalid json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - email too short",
			body: map[string]string{
				"email":    "",
				"password": "password123",
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unauthorized - user not found",
			body: map[string]string{
				"email":    "notfound@example.com",
				"password": "password123",
			},
			setupMock: func() {
				mockRepo.EXPECT().
					GetByEmail(gomock.Any(), "notfound@example.com").
					Return(entity.User{}, usecase.ErrNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "unauthorized - wrong password",
			body: map[string]string{
				"email":    "test@example.com",
				"password": "wrongpassword",
			},
			setupMock: func() {
				hashedPassword, _ := auth.HashPassword("password123")
				user := TestUser
				user.Password = hashedPassword
				mockRepo.EXPECT().
					GetByEmail(gomock.Any(), "test@example.com").
					Return(user, nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users/login", nil)
			if bodyMap, ok := tt.body.(map[string]string); ok {
				body, _ := json.Marshal(bodyMap)
				r = httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
			}

			handler.LoginUser(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, w.Body.String(), "access_token")
			}
		})
	}
}

func TestUserHandler_GetCurrentUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := mocks.NewMockUserRepository(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	handler := NewUserHandler(mockRepo, mockSessionRepo, "test-secret")

	tests := []struct {
		name           string
		setupMock      func() context.Context
		expectedStatus int
	}{
		{
			name: "success - authenticated user",
			setupMock: func() context.Context {
				mockRepo.EXPECT().
					GetByID(gomock.Any(), TestUser.ID).
					Return(TestUser, nil)
				ctx := context.WithValue(context.Background(), userIDKey, TestUser.ID)
				ctx = context.WithValue(ctx, roleKey, TestUser.Role)
				return ctx
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorized - missing token",
			setupMock: func() context.Context {
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupMock()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/me", nil).WithContext(ctx)

			handler.GetCurrentUser(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
