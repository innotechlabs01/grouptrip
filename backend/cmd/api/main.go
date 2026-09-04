// Command api boots the grouptrip backend HTTP server, wiring the repositories,
// payment provider, and HTTP routes.
package main

import (
	"log"
	"net/http"
	"os"

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

	// Payment provider. Requires Polar env (POLAR_ACCESS_TOKEN / base URL); a missing
	// token does not crash boot — contribution charges simply fail at request time.
	_ = payments.NewPolarClientFromEnv(http.DefaultClient)

	// HTTP server; the Polar webhook route is enabled because contribRepo is wired.
	// The webhook signature is verified with POLAR_WEBHOOK_SECRET; if it is unset the
	// webhook route fails closed (processes no events).
	srv := httptransport.NewServerWithWebhook(fundRepo, contribRepo, os.Getenv("POLAR_WEBHOOK_SECRET"))

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("api: grouptrip backend listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("api: serve: %v", err)
	}
}
