package githubc

import (
	"context"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub client to provide access to all methods
type Client struct {
	*github.Client
}

func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &Client{client}
}
