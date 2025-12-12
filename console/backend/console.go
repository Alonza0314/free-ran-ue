package backend

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/free-ran-ue/free-ran-ue/constant"
	"github.com/free-ran-ue/free-ran-ue/logger"
	"github.com/free-ran-ue/free-ran-ue/model"
	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

type jwt struct {
	secret    string
	expiresIn time.Duration
}

type console struct {
	router *gin.Engine

	server *http.Server

	username string
	password string

	port int

	jwt

	frontendFilePath string

	*logger.ConsoleLogger
}

func NewConsole(config *model.ConsoleConfig, logger *logger.ConsoleLogger) *console {
	c := &console{
		router: nil,

		username: config.Console.Username,
		password: config.Console.Password,

		port: config.Console.Port,

		jwt: jwt{
			secret:    config.Console.JWT.Secret,
			expiresIn: config.Console.JWT.ExpiresIn,
		},

		frontendFilePath: config.Console.FrontendFilePath,

		ConsoleLogger: logger,
	}

	c.router = util.NewGinRouter(constant.API_PREFIX_CONSOLE, c.initRoutes())

	c.router.NoRoute(c.returnPages())
	return c
}

func (cs *console) Start() {
	cs.ConsoleLog.Infoln("Starting console")

	cs.server = &http.Server{
		Addr:    ":" + strconv.Itoa(cs.port),
		Handler: cs.router,
	}

	go func() {
		if err := cs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cs.ConsoleLog.Errorf("Failed to start console: %v", err)
		}
	}()
	time.Sleep(500 * time.Millisecond)

	cs.ConsoleLog.Infoln("Console started")
}

func (cs *console) Stop() {
	cs.ConsoleLog.Infoln("Stopping console")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := cs.server.Shutdown(shutdownCtx); err != nil {
		cs.ConsoleLog.Errorf("Failed to stop console: %v", err)
	} else {
		cs.ConsoleLog.Infoln("Console stopped successfully")
	}
}

func (cs *console) returnPages() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet {

			destPath := filepath.Join(cs.frontendFilePath, c.Request.URL.Path)
			if _, err := os.Stat(destPath); err == nil {
				c.File(filepath.Clean(destPath))
				return
			}

			c.File(filepath.Clean("build/console/index.html"))
		} else {
			c.Next()
		}
	}
}

func (cs *console) initRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Console Login",
			Method:      http.MethodPost,
			Pattern:     "/login",
			HandlerFunc: cs.handleConsoleLogin,
		},
		{
			Name:        "Console Logout",
			Method:      http.MethodDelete,
			Pattern:     "/logout",
			HandlerFunc: cs.handleConsoleLogout,
		},
		{
			Name:        "Authenticate",
			Method:      http.MethodPost,
			Pattern:     "/authenticate",
			HandlerFunc: cs.handleAuthenticate,
		},
		{
			Name:        "Console GNB Info",
			Method:      http.MethodPost,
			Pattern:     "/gnb/info",
			HandlerFunc: cs.handleConsoleGnbInfo,
		},
		{
			Name:        "Console GNB UE NRDC Modify",
			Method:      http.MethodPost,
			Pattern:     "/gnb/ue/nrdc",
			HandlerFunc: cs.handleConsoleGnbUeNrdcModify,
		},
	}
}
