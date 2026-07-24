package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/eefenaxce/axce_blog/internal/config"
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewPostgres(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?pool_max_conns=%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.MaxConnections)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConnections)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Auto-migrate: create tables if they don't exist
	if err := autoMigrate(pool); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	// Run idempotent migrations for existing databases
	if err := runMigrations(pool); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Database{Pool: pool}, nil
}

func autoMigrate(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// List of all tables defined in schema.sql
	tables := []string{"users", "categories", "articles", "tags", "article_tags", "comments", "menus", "menu_items", "settings"}

	// Check if all tables exist
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1 AND table_schema = 'public')",
			table,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			// At least one table is missing, need to run migration
			goto runMigration
		}
	}

	// All tables exist, skip migration
	return nil

runMigration:

	// Read and execute schema.sql
	schemaPath := filepath.Join("sqlc", "schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = pool.Exec(ctx, string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	fmt.Println("Database tables created successfully")

	// Seed default settings
	seedPath := filepath.Join("sqlc", "seed.sql")
	seed, err := os.ReadFile(seedPath)
	if err != nil {
		// Seed file is optional, just log and continue
		fmt.Printf("Warning: seed file not found, skipping: %v\n", err)
		return nil
	}

	// Execute seed SQL using transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to begin transaction for seed: %v\n", err)
		return nil
	}
	defer tx.Rollback(ctx)

	// Split SQL by semicolon and execute each statement
	seedSQL := string(seed)
	var currentStmt strings.Builder
	inComment := false
	inString := false
	stringChar := rune(0)

	for i, r := range seedSQL {
		// Handle comments
		if r == '-' && i+1 < len(seedSQL) && seedSQL[i+1] == '-' {
			inComment = true
		}
		if inComment && r == '\n' {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		// Handle strings
		if !inString && (r == '\'' || r == '"') {
			inString = true
			stringChar = r
			currentStmt.WriteRune(r)
			continue
		}
		if inString && r == stringChar {
			// Check if it's escaped
			if i > 0 && seedSQL[i-1] != '\\' {
				inString = false
				stringChar = rune(0)
			}
			currentStmt.WriteRune(r)
			continue
		}
		if inString {
			currentStmt.WriteRune(r)
			continue
		}

		// Handle statement separator
		if r == ';' {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" {
				if _, err := tx.Exec(ctx, stmt); err != nil {
					fmt.Printf("Warning: failed to execute seed statement: %v\nStatement: %s\n", err, stmt)
					return nil
				}
			}
			currentStmt.Reset()
			continue
		}

		currentStmt.WriteRune(r)
	}

	// Execute last statement if exists
	if stmt := strings.TrimSpace(currentStmt.String()); stmt != "" {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			fmt.Printf("Warning: failed to execute seed statement: %v\nStatement: %s\n", err, stmt)
			return nil
		}
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Printf("Warning: failed to commit seed transaction: %v\n", err)
		return nil
	}

	fmt.Println("Database seeded with default settings")
	return nil
}

// runMigrations executes idempotent migration statements from migrations.sql.
func runMigrations(pool *pgxpool.Pool) error {
	ctx := context.Background()
	migrationPath := filepath.Join("sqlc", "migrations.sql")
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		// Migration file is optional
		return nil
	}
	return execSQLStatements(ctx, pool, string(sqlBytes))
}

// execSQLStatements splits a SQL script by semicolons and executes each statement.
func execSQLStatements(ctx context.Context, pool *pgxpool.Pool, sql string) error {
	var currentStmt strings.Builder
	inComment := false
	inString := false
	stringChar := rune(0)

	for i, r := range sql {
		if r == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			inComment = true
		}
		if inComment && r == '\n' {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		if !inString && (r == '\'' || r == '"') {
			inString = true
			stringChar = r
			currentStmt.WriteRune(r)
			continue
		}
		if inString && r == stringChar {
			if i > 0 && sql[i-1] != '\\' {
				inString = false
				stringChar = rune(0)
			}
			currentStmt.WriteRune(r)
			continue
		}
		if inString {
			currentStmt.WriteRune(r)
			continue
		}

		if r == ';' {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" {
				if _, err := pool.Exec(ctx, stmt); err != nil {
					return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
				}
			}
			currentStmt.Reset()
			continue
		}

		currentStmt.WriteRune(r)
	}

	if stmt := strings.TrimSpace(currentStmt.String()); stmt != "" {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
		}
	}
	return nil
}

func (d *Database) Close() {
	d.Pool.Close()
}

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(cfg config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

func (r *RedisClient) Close() error {
	return r.Client.Close()
}
