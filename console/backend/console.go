package backend

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	"github.com/free-ran-ue/free-ran-ue/v2/constant"
	"github.com/free-ran-ue/free-ran-ue/v2/logger"
	"github.com/free-ran-ue/free-ran-ue/v2/model"
	"github.com/free-ran-ue/util"
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

	gin.DefaultWriter, gin.DefaultErrorWriter = loggergo.NewGinWriter(logger.GinLog), loggergo.NewGinWriter(logger.GinLog)

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
			HandlerFunc: withLogging("Console Login", cs.LoginLog, cs.handleConsoleLogin),
		},
		{
			Name:        "Console Logout",
			Method:      http.MethodDelete,
			Pattern:     "/logout",
			HandlerFunc: withLogging("Console Logout", cs.LogoutLog, cs.handleConsoleLogout),
		},
		{
			Name:        "Authenticate",
			Method:      http.MethodPost,
			Pattern:     "/authenticate",
			HandlerFunc: withLogging("Authenticate", cs.AuthLog, cs.handleAuthenticate),
		},
		{
			Name:        "Console GNB Info",
			Method:      http.MethodPost,
			Pattern:     "/gnb/info",
			HandlerFunc: withLogging("Console GNB Info", cs.GnbLog, cs.handleConsoleGnbInfo),
		},
		{
			Name:        "Console GNB UE NRDC Modify",
			Method:      http.MethodPost,
			Pattern:     "/gnb/ue/nrdc",
			HandlerFunc: withLogging("Console GNB UE NRDC Modify", cs.GnbLog, cs.handleConsoleGnbUeNrdcModify),
		},
	}
}

// withLogging wraps a route's handler so the request is logged exactly once,
// after it completes, with the level derived from the response status code
// the handler wrote (via c.JSON/c.Status).
func withLogging(name string, lg loggergoModel.LoggerInterface, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c)

		status := c.Writer.Status()
		switch {
		case status >= http.StatusInternalServerError:
			lg.Errorf("%s failed (status %d) for %s", name, status, c.ClientIP())
		case status >= http.StatusBadRequest:
			lg.Warnf("%s failed (status %d) for %s", name, status, c.ClientIP())
		default:
			lg.Infof("%s succeeded (status %d) for %s", name, status, c.ClientIP())
		}
	}
}
