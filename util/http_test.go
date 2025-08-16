package util

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var testSendHttpRequestCases = []struct {
	uri              string
	method           string
	headers          map[string]string
	data             []byte
	expectedResponse map[string]interface{}
	expectedStatus   int
	expectedHeaders  map[string]string
	expectedError    bool
}{
	{
		uri:              "http://localhost:12345/api/test",
		method:           "GET",
		headers:          map[string]string{},
		data:             []byte{},
		expectedResponse: map[string]interface{}{"message": "test response"},
		expectedStatus:   http.StatusOK,
		expectedHeaders:  map[string]string{"Content-Type": "application/json; charset=utf-8"},
		expectedError:    false,
	},
	{
		uri:              "http://localhost:12345/api/test",
		method:           "POST",
		headers:          map[string]string{"Content-Type": "application/json"},
		data:             []byte(`{"message": "test request"}`),
		expectedResponse: map[string]interface{}{"message": "test response"},
		expectedStatus:   http.StatusOK,
		expectedHeaders:  map[string]string{"Content-Type": "application/json; charset=utf-8"},
		expectedError:    false,
	},
}

func TestSendHttpRequest(t *testing.T) {
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test response"})
	})
	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test response"})
	})

	srv := &http.Server{
		Addr:    "localhost:12345",
		Handler: router,
	}

	errCh := make(chan error, 1)
	ready := make(chan struct{})
	go func() {
		ready <- struct{}{}
		errCh <- srv.ListenAndServe()
	}()

	<-ready

	time.Sleep(100 * time.Millisecond)

	for _, testCase := range testSendHttpRequestCases {
		t.Run(testCase.method+" "+testCase.uri, func(t *testing.T) {
			response, err := SendHttpRequest(testCase.uri, testCase.method, testCase.headers, testCase.data)
			if (err != nil) != testCase.expectedError {
				t.Fatalf("expected error: %v, got: %v", testCase.expectedError, err)
			}

			if !testCase.expectedError {
				if response.StatusCode != testCase.expectedStatus {
					t.Errorf("expected status code: %v, got: %v", testCase.expectedStatus, response.StatusCode)
				}

				for key, expectedValue := range testCase.expectedHeaders {
					if actualValue := response.Headers.Get(key); actualValue != expectedValue {
						t.Errorf("expected header %s: %v, got: %v", key, expectedValue, actualValue)
					}
				}

				var actualResponse map[string]interface{}
				if err := json.Unmarshal(response.Body, &actualResponse); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}

				for key, expectedValue := range testCase.expectedResponse {
					if actualValue, ok := actualResponse[key]; !ok || actualValue != expectedValue {
						t.Errorf("expected response[%s]: %v, got: %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

	if err := <-errCh; err != nil && err != http.ErrServerClosed {
		t.Fatalf("Server error: %v", err)
	}
}
