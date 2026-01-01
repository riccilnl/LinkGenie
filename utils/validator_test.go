package utils

import (
	"testing"

	"ai-bookmark-service/models"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:        "URL with https protocol",
			input:       "https://example.com",
			expected:    "https://example.com",
			shouldError: false,
		},
		{
			name:        "URL with http protocol",
			input:       "http://example.com",
			expected:    "http://example.com",
			shouldError: false,
		},
		{
			name:        "URL without protocol",
			input:       "example.com",
			expected:    "https://example.com",
			shouldError: false,
		},
		{
			name:        "URL with www without protocol",
			input:       "www.example.com",
			expected:    "https://www.example.com",
			shouldError: false,
		},
		{
			name:        "URL with path without protocol",
			input:       "example.com/path/to/page",
			expected:    "https://example.com/path/to/page",
			shouldError: false,
		},
		{
			name:        "URL with query params without protocol",
			input:       "example.com?foo=bar",
			expected:    "https://example.com?foo=bar",
			shouldError: false,
		},
		{
			name:        "URL with whitespace",
			input:       "  example.com  ",
			expected:    "https://example.com",
			shouldError: false,
		},
		{
			name:        "Empty URL",
			input:       "",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "URL with unsupported protocol",
			input:       "ftp://example.com",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Invalid URL",
			input:       "not a valid url",
			expected:    "https://not a valid url",
			shouldError: true,
		},
		{
			name:        "URL with port",
			input:       "example.com:8080",
			expected:    "https://example.com:8080",
			shouldError: false,
		},
		{
			name:        "URL with subdomain",
			input:       "api.example.com",
			expected:    "https://api.example.com",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeURL(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("For input '%s', expected '%s', but got '%s'", tt.input, tt.expected, result)
				}
			}
		})
	}
}

func TestValidateBookmarkCreate(t *testing.T) {
	tests := []struct {
		name        string
		bookmark    *models.BookmarkCreate
		shouldError bool
		expectedURL string
	}{
		{
			name: "Valid bookmark with https URL",
			bookmark: &models.BookmarkCreate{
				URL:   "https://example.com",
				Title: "Example",
			},
			shouldError: false,
			expectedURL: "https://example.com",
		},
		{
			name: "Valid bookmark without protocol",
			bookmark: &models.BookmarkCreate{
				URL:   "example.com",
				Title: "Example",
			},
			shouldError: false,
			expectedURL: "https://example.com",
		},
		{
			name: "Valid bookmark with www without protocol",
			bookmark: &models.BookmarkCreate{
				URL:   "www.example.com",
				Title: "Example",
			},
			shouldError: false,
			expectedURL: "https://www.example.com",
		},
		{
			name: "Empty URL",
			bookmark: &models.BookmarkCreate{
				URL:   "",
				Title: "Example",
			},
			shouldError: true,
		},
		{
			name: "Invalid URL",
			bookmark: &models.BookmarkCreate{
				URL:   "not a valid url",
				Title: "Example",
			},
			shouldError: true,
		},
		{
			name: "Title too long",
			bookmark: &models.BookmarkCreate{
				URL:   "example.com",
				Title: string(make([]byte, 201)),
			},
			shouldError: true,
		},
		{
			name: "Description too long",
			bookmark: &models.BookmarkCreate{
				URL:         "example.com",
				Title:       "Example",
				Description: string(make([]byte, 1001)),
			},
			shouldError: true,
		},
		{
			name: "Too many tags",
			bookmark: &models.BookmarkCreate{
				URL:      "example.com",
				Title:    "Example",
				TagNames: make([]string, 51), // 现在上限是 50
			},
			shouldError: true,
		},
		{
			name: "Valid bookmark with tags",
			bookmark: &models.BookmarkCreate{
				URL:      "example.com",
				Title:    "Example",
				TagNames: []string{"tech", "golang", "编程"},
			},
			shouldError: false,
			expectedURL: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBookmarkCreate(tt.bookmark)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectedURL != "" && tt.bookmark.URL != tt.expectedURL {
					t.Errorf("Expected URL to be normalized to '%s', but got '%s'", tt.expectedURL, tt.bookmark.URL)
				}
			}
		})
	}
}
