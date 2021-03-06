package types

import (
	"context"
	"io"
	"net/http"
)

type TagCounts struct {
	Tag   string
	Count int
}

type ReadableFile interface {
	io.ReadCloser
}

type WriteableFile interface {
	io.WriteCloser
}

type Storage interface {
	ReadFile(context.Context, string) (ReadableFile, error)
	WriteFile(context.Context, string) (WriteableFile, error)
	Open(string) (http.File, error)
	Delete(string) error
}

type Session struct {
	Username       string `json:"username"`
	ExpirationTime int64  `json:"expirationTime"`
}

type User struct {
	// AvatarID is the post ID of the author's avatar.
	AvatarID int64 `json:"avatarID"`
	// Owner allows for making other users admins.
	Owner bool `json:"owner"`
	// Admin allows full access to the site functions.
	Admin bool `json:"admin"`
	// Username of the user.
	Username string `json:"username"`
	// Description of the user, used in the user view page.
	Description string `json:"description"`
	// Posts is a list of all the IDs of the posts that the user
	// created, also used in the user view page for a preview.
	Posts []int64 `json:"posts"`
	// Theme is a string of the theme.
	Theme string `json:"theme"`
}

type Post struct {
	// Filename of the file.
	// TODO: Deprecate this field as filename == ID.
	Filename string `json:"filename"`
	// FileExtension is the extension of the file, inferred from mime type.
	FileExtension string `json:"ext"`
	// Description used when viewing the post.
	Description string `json:"description"`
	// Tags are a list of tags used when searching for posts.
	Tags []string `json:"tags"`
	// PostID specifies the ID of the post.
	PostID int64 `json:"postID"`
	// PosterID is the user ID of the user who posted this.
	Poster string `json:"poster"`
	// CreatedAt is the Unix timestamp of when this post was posted.
	CreatedAt int64 `json:"timestamp"`
	// MimeType is the MIME type of the post file.
	MimeType string `json:"mimetype"`
}
