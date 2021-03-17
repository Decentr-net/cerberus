package test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/sirupsen/logrus"

	logging "github.com/Decentr-net/logrus/context"
)

// NewAPITestParameters returns a logger's buffer, ResponseRecorder and Request.
// Panics if any errors occurred.
func NewAPITestParameters(method string, uri string, body []byte) (*bytes.Buffer, *httptest.ResponseRecorder, *http.Request) {
	l := logrus.New()
	b := bytes.NewBufferString("")
	l.SetLevel(logrus.TraceLevel)
	l.SetOutput(b)

	r, err := http.NewRequestWithContext(
		logging.WithLogger(context.Background(), l),
		method,
		fmt.Sprintf("http://localhost/%s", uri),
		bytes.NewReader(body),
	)
	if err != nil {
		panic(err)
	}

	return b, httptest.NewRecorder(), r
}
