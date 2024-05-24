package main

import (
	"context"
	"flag"
	"hash/fnv"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/discovertomorrow/progai-middleware/pkg/handler"
	"github.com/discovertomorrow/progai-middleware/pkg/llamacpp"
	"github.com/discovertomorrow/progai-middleware/pkg/logging"
	"github.com/discovertomorrow/progai-middleware/pkg/session"
	"github.com/discovertomorrow/progai-middleware/pkg/usage"
)

func main() {

	// flags
	endpointFlag := flag.String(
		"endpoint",
		os.Getenv("AI_ENDPOINT"),
		"URL of the completion endpoint",
	)
	envSlots, _ := strconv.Atoi(os.Getenv("AI_SLOTS"))
	slotsFlag := flag.Int("slots", envSlots, "number of slots in Llama.cpp")
	tmplFlag := flag.String("template", os.Getenv("AI_TEMPLATE"), "go template for messages")
	stopFlag := flag.String("stop", os.Getenv("AI_STOP"), "stop tokens, comma separated")
	debugFlag := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()

	// setup logger
	logLevel := slog.LevelInfo
	if *debugFlag {
		logLevel = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	logHandler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(logHandler)

	// http.Handler
	h := llamacpp.NewLlamacppHandler(
		true, // lineByLine
		[]handler.Endpoint{{Endpoint: *endpointFlag, Parallel: *slotsFlag}},
	)

	sessionMiddleware := session.Middleware(getSessionData)

	// usage
	usageMiddleware := usage.UsageTracker(
		llamacpp.NewLlamacppUsageUpdater(),
		func(ctx context.Context, u usage.Usage) {
			logging.FromContext(ctx).Info("Logging Usage", "usage", u)
		},
	)

	// serve
	mux := http.NewServeMux()
	mux.Handle(
		"POST /",
		logging.Middleware(logger.With("handle", "/"))(
			sessionMiddleware(
				session.TokenLimiter()(
					handler.Limiter(*slotsFlag)(h)))),
	)

	if *tmplFlag != "" && *stopFlag != "" {
		// http chat handler
		ch := llamacpp.NewLlamacppChatHandler(
			logger,
			true, // lineByLine
			[]handler.Endpoint{{Endpoint: *endpointFlag, Parallel: *slotsFlag}},
			*tmplFlag,
			strings.Split(*stopFlag, ","),
		)

		mux.Handle(
			"POST /v1/chat/completions",
			logging.Middleware(logger.With("handle", "/v1/chat/completions"))(
				sessionMiddleware(
					usageMiddleware(
						session.TokenLimiter()(
							handler.Limiter(*slotsFlag)(ch))))),
		)
	}

	logger.Info("Starting to serve")
	http.ListenAndServe("0.0.0.0:8080", mux)
}

func getSessionData(r *http.Request) (session.SessionData, bool) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(token) < 2 {
		return session.SessionData{}, false
	}
	return session.SessionData{
		TokenID:               int(hashStringToRange(token, 1000)),
		UserID:                token,
		TokenConcurrencyLimit: 1,
	}, true
}

func hashStringToRange(s string, rangeMax uint32) uint32 {
	hasher := fnv.New32a()
	_, err := hasher.Write([]byte(s))
	if err != nil {
		panic(err)
	}
	hash := hasher.Sum32()
	return hash % rangeMax
}
