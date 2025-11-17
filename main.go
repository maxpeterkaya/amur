package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Error().Err(err).Msg("Error loading .env file")
	}

	ConfigNew()

	NewAsynqClient()

	if err := CreateEssentialFolders(); err != nil {
		log.Error().Err(err).Msg("Error creating folders")
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	e := echo.New()

	// Middleware
	e.IPExtractor = echo.ExtractIPFromXFFHeader()
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(Config.Cors))
	e.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: func(c echo.Context) bool {
			// skip for all routes but metrics
			if c.Path() != "/metrics" {
				return true
			}

			return false
		},
		Validator: func(username, password string, c echo.Context) (bool, error) {
			if subtle.ConstantTimeCompare([]byte(username), []byte(Config.Prometheus.Username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(password), []byte(Config.Prometheus.Password)) == 1 {
				return true, nil
			}

			return false, nil
		},
	}))
	e.Use(echoprometheus.NewMiddleware("amur"))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:           true,
		LogStatus:        true,
		LogError:         true,
		LogHost:          true,
		LogLatency:       true,
		LogMethod:        true,
		LogContentLength: true,
		LogProtocol:      true,
		LogReferer:       true,
		LogUserAgent:     true,
		LogRemoteIP:      true,
		LogRequestID:     true,
		LogResponseSize:  true,
		LogURIPath:       true,
		LogRoutePath:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			ext := HasExtension(v.URI)
			route := ""

			if !ext {
				route = v.URI
			}

			if v.Error != nil {
				log.Error().
					Err(v.Error).
					Str("URI", v.URI).
					Int("status", v.Status).
					Str("status_text", http.StatusText(v.Status)).
					Str("method", v.Method).
					Str("remote_ip", v.RemoteIP).
					Str("host", v.Host).
					Str("uri", v.URI).
					Str("protocol", v.Protocol).
					Str("referer", v.Referer).
					Str("user_agent", v.UserAgent).
					Str("id", v.RequestID).
					Int("latency", int(v.Latency.Nanoseconds())).
					Str("latency_human", v.Latency.String()).
					Int("bytes_in", int(c.Request().ContentLength)).
					Str("bytes_in_human", ByteCountSI(c.Request().ContentLength)).
					Int("bytes_out", int(v.ResponseSize)).
					Str("bytes_out_human", ByteCountSI(v.ResponseSize)).
					Str("route", route).
					Str("server_version", version).
					Str("server_commit", commit).
					Str("server_build_date", date).
					Msg("error")
			} else {
				log.Info().
					Str("URI", v.URI).
					Int("status", v.Status).
					Str("status_text", http.StatusText(v.Status)).
					Str("method", v.Method).
					Str("remote_ip", v.RemoteIP).
					Str("host", v.Host).
					Str("uri", v.URI).
					Str("protocol", v.Protocol).
					Str("referer", v.Referer).
					Str("user_agent", v.UserAgent).
					Str("id", v.RequestID).
					Int("latency", int(v.Latency.Nanoseconds())).
					Str("latency_human", v.Latency.String()).
					Int("bytes_in", int(c.Request().ContentLength)).
					Str("bytes_in_human", ByteCountSI(c.Request().ContentLength)).
					Int("bytes_out", int(v.ResponseSize)).
					Str("bytes_out_human", ByteCountSI(v.ResponseSize)).
					Str("route", route).
					Str("server_version", version).
					Str("server_commit", commit).
					Str("server_build_date", date).
					Msg("request")
			}

			return nil
		},
	}))

	// Routes
	e.RouteNotFound("/*", ServeFile)
	e.POST("/upload", UploadFile)
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/health", func(c echo.Context) error {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		log.Info().Str("version", version).Str("goroutines", strconv.Itoa(runtime.NumGoroutine())).Str("cpu", strconv.Itoa(runtime.NumCPU())).Str("allocated_memory", ByteCountSI(int64(mem.TotalAlloc))).Str("memory_allocations", ByteCountSI(int64(mem.Mallocs))).Msg("Health check")
		return c.String(http.StatusOK, "ok")
	})

	log.Info().Str("version", version).Str("commit", commit).Str("date", date).Msg("")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go WatchFolder()
	go CronInit()

	if Config.UseRedis {
		go func() {
			NewAsynqServer()
		}()
	}

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", Config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Msg("error starting server")
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("error shutting down server")
	} else {
		log.Info().Msg("shutting down server")
	}
}
