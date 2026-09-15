// Command idp runs the UC-03 Internal Developer Platform: schema migration,
// fixture import, the web/API server and the background Deployment Worker.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pr3s3nt/final_idp/uc03/internal/config"
	"github.com/pr3s3nt/final_idp/uc03/internal/fixtures"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/uc03/internal/persistence"
)

const usage = `usage: idp <command>

commands:
  migrate                      apply database migrations
  import-fixtures [dir]        import catalog, application versions and configurations
  serve                        run the web UI and JSON API
  worker                       run the background Deployment Worker
  fail-orphaned-job <id> <msg> mark a deployment whose worker died as FAILED`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1], os.Args[2:]); err != nil {
		log.Fatalf("idp %s: %v", os.Args[1], err)
	}
}

func run(ctx context.Context, cmd string, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	switch cmd {
	case "migrate":
		applied, err := db.Migrate(ctx)
		for _, name := range applied {
			log.Printf("applied %s", name)
		}
		return err
	case "import-fixtures":
		dir := cfg.FixturesDir
		if len(args) > 0 {
			dir = args[0]
		}
		secrets, err := secretstore.NewEncryptedFile(filepath.Join(cfg.DataDir, "secrets"), cfg.SecretKey)
		if err != nil {
			return err
		}
		im := &fixtures.Importer{
			Apps:    &persistence.ApplicationRepository{DB: db},
			Configs: &persistence.EnvironmentConfigurationRepository{DB: db},
			Catalog: &persistence.ResourceDefinitionCatalog{DB: db},
			Secrets: secrets,
			Log:     log.Printf,
		}
		return im.ImportDirectory(ctx, dir)
	case "secret-put":
		// Platform administration: store a value (for example the connection
		// record of a shared resource) and print its opaque reference.
		if len(args) != 2 {
			return fmt.Errorf("usage: idp secret-put <name> <file>")
		}
		secrets, err := secretstore.NewEncryptedFile(filepath.Join(cfg.DataDir, "secrets"), cfg.SecretKey)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		ref, err := secrets.Put(ctx, args[0], body)
		if err == nil {
			fmt.Println(ref)
		}
		return err
	case "serve", "worker", "fail-orphaned-job":
		return runPlatform(ctx, cfg, db, cmd, args)
	default:
		return fmt.Errorf("unknown command %q\n%s", cmd, usage)
	}
}
