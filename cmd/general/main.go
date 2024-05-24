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

	// setup logger
	logger := slog.Default()

	// flags
	endpointFlag := flag.String(
		"endpoint",
		os.Getenv("AI_ENDPOINT"),
		"URL of the endpoint",
	)
	envSlots, _ := strconv.Atoi(os.Getenv("AI_SLOTS"))
	slotsFlag := flag.Int("slots", envSlots, "number of slots in Llama.cpp")
	flag.Parse()

	// http.Handler
	h := handler.NewDefaultHandler(false, handler.Endpoint{Endpoint: *endpointFlag, Parallel: *slotsFlag})

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
				usageMiddleware(
					session.TokenLimiter()(
						handler.Limiter(*slotsFlag)(h))))),
	)

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
