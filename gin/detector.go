package gin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	botdetector "github.com/krakendio/krakend-botdetector/v2"
	krakend "github.com/krakendio/krakend-botdetector/v2/krakend"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	krakendgin "github.com/luraproject/lura/v2/router/gin"
)

const logPrefix = "[SERVICE: Gin][Botdetector]"

// Register checks the configuration and, if required, registers a bot detector middleware at the gin engine
func Register(cfg config.ServiceConfig, l logging.Logger, engine *gin.Engine) {
	detectorCfg, err := krakend.ParseConfig(cfg.ExtraConfig)
	if err == krakend.ErrNoConfig {
		return
	}
	if err != nil {
		l.Warning(logPrefix, err.Error())
		return
	}
	d, err := botdetector.New(detectorCfg)
	if err != nil {
		l.Warning(logPrefix, "Unable to create the bot detector:", err.Error())
		return
	}

	l.Debug(logPrefix, "The bot detector has been registered successfully")
	engine.Use(middleware(d, l))
}

// New checks the configuration and, if required, wraps the handler factory with a bot detector middleware
func New(hf krakendgin.HandlerFactory, l logging.Logger) krakendgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		next := hf(cfg, p)
		logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][Botdetector]"

		detectorCfg, err := krakend.ParseConfig(cfg.ExtraConfig)
		if err == krakend.ErrNoConfig {
			return next
		}
		if err != nil {
			l.Warning(logPrefix, err.Error())
			return next
		}

		d, err := botdetector.New(detectorCfg)
		if err != nil {
			l.Warning(logPrefix, "Unable to create the bot detector:", err.Error())
			return next
		}

		l.Debug(logPrefix, "The bot detector has been registered successfully")
		return handler(d, next, l)
	}
}

func middleware(f botdetector.DetectorFunc, l logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if f(c.Request) {
			l.Error(logPrefix, errBotRejected)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

func handler(f botdetector.DetectorFunc, next gin.HandlerFunc, l logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if f(c.Request) {
			l.Error(logPrefix, errBotRejected)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		next(c)
	}
}

var errBotRejected = errors.New("bot rejected")
