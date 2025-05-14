package fixtures

import (
	"net/http"
	"net/http/httptest"
)

func FixtureServer() *httptest.Server {
	return httptest.NewServer(http.FileServerFS(EmbeddedFixtures))
}

func FixtureTLSServer() *httptest.Server {
	return httptest.NewServer(http.FileServerFS(EmbeddedFixtures))
}
