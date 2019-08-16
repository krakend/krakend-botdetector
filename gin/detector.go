package gin

import (
	"errors"
	"net/http"

	botdetector "github.com/devopsfaith/krakend-botdetector"
	krakend "github.com/devopsfaith/krakend-botdetector/krakend"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"
)

// Register checks the configuration and, if required, registers a bot detector middleware at the gin engine
func Register(cfg config.ServiceConfig, l logging.Logger, engine *gin.Engine) {
	detectorCfg, err := krakend.ParseConfig(cfg.ExtraConfig)
	if err == krakend.ErrNoConfig {
		l.Debug("botdetector middleware: ", err.Error())
		return
	}
	if err != nil {
		l.Warning("botdetector middleware: ", err.Error())
		return
	}
	d, err := botdetector.New(detectorCfg)
	if err != nil {
		l.Warning("botdetector middleware: unable to createt the LRU detector:", err.Error())
		return
	}
	engine.Use(middleware(d))
}

// New checks the configuration and, if required, wraps the handler factory with a bot detector middleware
func New(hf krakendgin.HandlerFactory, l logging.Logger) krakendgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		next := hf(cfg, p)

		detectorCfg, err := krakend.ParseConfig(cfg.ExtraConfig)
		if err == krakend.ErrNoConfig {
			l.Debug("botdetector: ", err.Error())
			return next
		}
		if err != nil {
			l.Warning("botdetector: ", err.Error())
			return next
		}

		d, err := botdetector.New(detectorCfg)
		if err != nil {
			l.Warning("botdetector: unable to create the LRU detector:", err.Error())
			return next
		}
		return handler(d, next)
	}
}

func middleware(f botdetector.DetectorFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if f(c.Request) {
			c.AbortWithError(http.StatusForbidden, errBotRejected)
			return
		}

		c.Next()
	}
}

func handler(f botdetector.DetectorFunc, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if f(c.Request) {
			c.AbortWithError(http.StatusForbidden, errBotRejected)
			return
		}

		next(c)
	}
}

var errBotRejected = errors.New("bot rejected")
