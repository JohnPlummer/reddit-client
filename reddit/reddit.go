package reddit

// RedditClient embeds Client, automatically exposing its methods.
type RedditClient struct {
	Auth    *Auth
	*Client // Embedding Client removes redundancy
}

// NewClient initializes a Reddit API client using client credentials flow.
func NewClient(clientID, clientSecret string) (*RedditClient, error) {
	auth := &Auth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if err := auth.Authenticate(); err != nil {
		return nil, err
	}

	client := &Client{Auth: auth}
	return &RedditClient{Auth: auth, Client: client}, nil
}
