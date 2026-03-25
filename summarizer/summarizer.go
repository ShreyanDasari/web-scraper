package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ollama/ollama/api"
	"github.com/redis/go-redis/v9" // Add this import
)

func main() {
	ctx := context.Background()

	// 1. Database Setup
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		connStr = "postgres://postgres:abc@123@localhost:5433/web-scraper"
	}
	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatal("DB Connection failed:", err)
	}
	defer db.Close(ctx)

	// 2. Redis Setup (New!)
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"), // Set this to "redis:6379" in your .env
	})

	// 3. Ollama Client Setup
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🚀 Summarizer Worker is Online and Waiting for Tasks...")

	for {
		// --- STEP A: WAIT FOR TASK FROM REDIS ---
		// BRPop "blocks" (waits) until an ID appears in the 'scrape_tasks' list.
		// Result format: [list_name, value] -> ["scrape_tasks", "12"]
		result, err := rdb.BRPop(ctx, 0, "scrape_tasks").Result()
		if err != nil {
			log.Printf("⚠️ Redis error: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		articleID := result[1]
		fmt.Printf("📥 Received Task for ID: %s\n", articleID)

		// --- STEP B: FETCH CONTENT FROM DB ---
		var content string
		query := "SELECT raw_content FROM scraped_pages WHERE id = $1"
		err = db.QueryRow(ctx, query, articleID).Scan(&content)
		if err != nil {
			log.Printf("❌ DB Fetch Error for ID %s: %v", articleID, err)
			continue
		}

		// --- STEP C: PROCESS WITH AI ---
		fmt.Printf("🤖 Processing ID %s with Ollama...\n", articleID)
		req := &api.GenerateRequest{
			Model:  "smollm2:latest",
			Prompt: "Summarize the following text in exactly 120 words: " + content,
			Stream: new(bool),
		}

		var summaryText string
		respFunc := func(resp api.GenerateResponse) error {
			summaryText = resp.Response
			return nil
		}

		if err := client.Generate(ctx, req, respFunc); err != nil {
			log.Printf("❌ AI Error for ID %s: %v. Re-queueing...", articleID, err)
			// PATH 1: If AI fails, push it back to the queue to try again later
			rdb.LPush(ctx, "scrape_tasks", articleID)
			continue
		}

		// --- STEP D: UPDATE THE DB ---
		updateQuery := "UPDATE scraped_pages SET summary = $1, is_summarized = true WHERE id = $2"
		_, err = db.Exec(ctx, updateQuery, summaryText, articleID)
		if err != nil {
			log.Printf("❌ Update failed for ID %s: %v", articleID, err)
		} else {
			fmt.Printf("✅ Summary saved for ID %s\n", articleID)
		}
	}
}