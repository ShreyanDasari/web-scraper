package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-shiori/go-readability"
	"github.com/gocolly/colly"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func initDatabase(db *pgx.Conn) error {
	query := `
    CREATE TABLE IF NOT EXISTS scraped_pages (
        id SERIAL PRIMARY KEY,
        url TEXT UNIQUE NOT NULL,
        raw_content TEXT,
        summary TEXT,
        is_summarized BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	_, err := db.Exec(context.Background(), query)
	return err
}
func main() {
	// --- DATABASE SETUP ---
	ctx := context.Background()
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		connStr = "postgres://postgres:abc@123@localhost:5433/web-scraper"
	}
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx) // Closes connection when main() finishes
	// 1. Define a list of URLs to scrape
	// ... after connecting to the DB ...
	err = initDatabase(conn)
	if err != nil {
		log.Fatalf("Could not create tables: %v", err)
	}
	log.Println("✅ Database tables are ready!")
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"), // Redis server address
	})

	urls := []string{
		"https://www.livemint.com/news/india/cabinet-extends-ivfrt-scheme-for-5-years-to-boost-immigration-and-visa-processing-details-here-11774441356146.html",
		"https://www.aljazeera.com/features/2026/3/25/amid-us-israeli-attacks-people-in-iran-struggle-to-survive-ailing-economy",
		"https://www.aljazeera.com/sports/2026/3/24/when-are-uefas-world-cup-2026-playoffs-and-which-nations-are-involved"}

	c := colly.NewCollector()

	// Set User-Agent to avoid being blocked
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

	// 2. Safety: Don't hammer the server too fast
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*livemint.com*",
		Parallelism: 2,
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("\n--- Processing: %s ---\n", r.Request.URL)

		// 3. Use the response's internal Reader and URL snapshot
		reader := strings.NewReader(string(r.Body))

		// We pass r.Request.URL here so readability knows exactly which page it's on
		article, err := readability.FromReader(reader, r.Request.URL)
		if err != nil {
			fmt.Printf("Failed to parse %s: %v\n", r.Request.URL, err)
			return
		}

		fmt.Println("TITLE:", article.Title)
		fmt.Println("CONTENT PREVIEW:", article.TextContent)
		var lastInsertedID int
		query := `
			INSERT INTO scraped_pages (url, raw_content, is_summarized) 
			VALUES ($1, $2, $3)
			ON CONFLICT (url) DO NOTHING
			RETURNING id;`

		err = conn.QueryRow(ctx, query,
			r.Request.URL.String(), // $1
			article.TextContent,    // $2
			false,                  // $3 (Default state)
		).Scan(&lastInsertedID)

		if err != nil {
			// If the error is 'no rows in result set', it just means ON CONFLICT happened
			if err == pgx.ErrNoRows {
				fmt.Println("⏭️ Skipping Redis: URL already exists in database.")
			} else {
				fmt.Printf("❌ Database error: %v\n", err)
			}
			return
		}
		err = rdb.LPush(ctx, "scrape_tasks", lastInsertedID).Err()
		if err != nil {
			log.Printf("⚠️ Redis Queue Error: %v", err)
		} else {
			fmt.Printf("🚀 Successfully saved & queued ID: %d\n", lastInsertedID)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error on %s: %v\n", r.Request.URL, err)
	})

	// 4. Start the loop
	for _, u := range urls {
		c.Visit(u)
	}
}
