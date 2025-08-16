package util_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Alonza0314/free-ran-ue/util"
)

var testGinRouterCases = util.Routes{
	util.Route{
		Name:    "TestRoute",
		Method:  http.MethodGet,
		Pattern: "/test",
		HandlerFunc: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		},
	},
	util.Route{
		Name:    "PostRoute",
		Method:  http.MethodPost,
		Pattern: "/post",
		HandlerFunc: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "post"})
		},
	},
}

func TestNewGinRouter(t *testing.T) {
	router := util.NewGinRouter(util.GNB_API_PREFIX, testGinRouterCases)
	assert.NotNil(t, router)

	for _, testCase := range testGinRouterCases {
		t.Run(testCase.Name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(testCase.Method, string(util.GNB_API_PREFIX)+testCase.Pattern, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestEmptyRoutes(t *testing.T) {
	router := util.NewGinRouter(util.GNB_API_PREFIX, util.Routes{})
	assert.NotNil(t, router)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/gnb/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
