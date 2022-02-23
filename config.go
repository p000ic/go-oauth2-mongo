package mongo

// Config mongodb configuration parameters
type Config struct {
	URL           string
	DB            string
	AuthMechanism string
	Username      string
	Password      string
	Source        string
}

// NewConfig create mongodb configuration
func NewConfig(url, db, username, password, authsource string) *Config {
	return &Config{
		URL:           url,
		DB:            db,
		AuthMechanism: "SCRAM-SHA-1",
		Username:      username,
		Password:      password,
		Source:        authsource,
	}
}
