package batch

// Config holds the run configuration.
type Config struct {
	URL         string
	Connections int
	Verbose     bool
	Headers     []string
	BearerToken string
	OAuth       OAuthConfig
	QueryFile   string
	InputFile   string
	OutputFile  string
	ErrorFile   string
}

// OAuthConfig holds the credentials for the OAuth 2.0 client credentials flow.
type OAuthConfig struct {
	URL          string
	ClientID     string
	ClientSecret string
	Scope        string
}
