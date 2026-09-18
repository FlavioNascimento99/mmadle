package store

import "os"

// lookupTestDBURL keeps env access in one place for tests.
func lookupTestDBURL() string {
	return os.Getenv("TEST_DATABASE_URL")
}
