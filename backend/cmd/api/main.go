// Command api boots the grouptrip backend HTTP server, wiring the repositories,
// payment provider, and HTTP routes.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/frg/grouptrip/internal/application/authservice"
	"github.com/frg/grouptrip/internal/application/commands"
	"github.com/frg/grouptrip/internal/application/events"
	"github.com/frg/grouptrip/internal/application/queries"
	"github.com/frg/grouptrip/internal/infrastructure/authrepo"
	"github.com/frg/grouptrip/internal/infrastructure/contribrepo"
	"github.com/frg/grouptrip/internal/infrastructure/fundrepo"
	"github.com/frg/grouptrip/internal/infrastructure/payments"
	httptransport "github.com/frg/grouptrip/internal/interfaces/http"
)

func main() {
	// Open the database (Turso remote when configured, local file otherwise).
	db, err := fundrepo.OpenTurso()
	if err != nil {
		log.Fatalf("api: open database: %v", err)
	}
	defer db.Close()

	fundRepo := fundrepo.NewSQLiteRepo(db)
	contribRepo := contribrepo.NewSQLiteContribRepo(db)

	if err := fundRepo.Migrate(); err != nil {
		log.Fatalf("api: migrate fund repo: %v", err)
	}
	if err := contribRepo.Migrate(); err != nil {
		log.Fatalf("api: migrate contribution repo: %v", err)
	}

	// Auth
	authRepo := authrepo.NewSQLiteAuthRepo(db)
	if err := authRepo.Migrate(); err != nil {
		log.Fatalf("api: migrate auth repo: %v", err)
	}
	jwtSecret := os.Getenv("AUTH_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-min-32-bytes-change-me!!"
		log.Printf("api: AUTH_JWT_SECRET not set, using dev secret")
	}
	authSvc := authservice.NewSessionService(authRepo, []byte(jwtSecret))

	// Payment provider. Requires Polar env (POLAR_ACCESS_TOKEN / base URL); a missing
	// token does not crash boot — contribution charges simply fail at request time.
	polar := payments.NewPolarClientFromEnv(http.DefaultClient)

	// Application commands and queries wired to repositories.
	contributeCmd := &commands.ContributeCommand{
		Funds:    fundRepo,
		Contrs:   contribRepo,
		Payments: polar,
		Events:   events.NoopSink{},
	}
	progressQuery := &queries.GetFundProgress{Funds: fundRepo}

	// HTTP server; the Polar webhook route is enabled because contribRepo is wired.
	// The webhook signature is verified with POLAR_WEBHOOK_SECRET; if it is unset the
	// webhook route fails closed (processes no events).
	srv := httptransport.NewServerWithAuth(
		fundRepo,
		contribRepo,
		os.Getenv("POLAR_WEBHOOK_SECRET"),
		contributeCmd,
		progressQuery,
		authSvc,
	)

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("api: grouptrip backend listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("api: serve: %v", err)
	}
}
