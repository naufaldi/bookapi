package profile

import (
	"bookapi/internal/user"
)

type Stats struct {
	BooksRead     int     `json:"books_read"`
	RatingsCount  int     `json:"ratings_count"`
	AverageRating float64 `json:"average_rating"`
}

type Profile struct {
	User  user.User `json:"user"`
	Stats Stats     `json:"stats"`
}

type UpdateCommand struct {
	Username           *string `json:"username"`
	Bio                *string `json:"bio"`
	Location           *string `json:"location"`
	Website            *string `json:"website"`
	IsPublic           *bool   `json:"is_public"`
	ReadingPreferences []byte  `json:"reading_preferences"`
}

func (c *UpdateCommand) ToMap() map[string]any {
	updates := make(map[string]any)
	if c.Username != nil {
		updates["username"] = *c.Username
	}
	if c.Bio != nil {
		updates["bio"] = *c.Bio
	}
	if c.Location != nil {
		updates["location"] = *c.Location
	}
	if c.Website != nil {
		updates["website"] = *c.Website
	}
	if c.IsPublic != nil {
		updates["is_public"] = *c.IsPublic
	}
	if c.ReadingPreferences != nil {
		updates["reading_preferences"] = c.ReadingPreferences
	}
	return updates
}
