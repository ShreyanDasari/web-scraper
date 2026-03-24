package main

import (
	"context"
	"fmt"
	"log"
	"time" // Added for the sleep timer

	"github.com/jackc/pgx/v5"
	"github.com/ollama/ollama/api"
)

func main() {
	ctx := context.Background()

	// 1. Connect to DB (OUTSIDE the loop - only do this once!)
	connStr := "postgres://postgres:abc@123@localhost:5433/web-scraper"
	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatal("DB Connection failed:", err)
	}
	defer db.Close(ctx)

	// 2. Setup Ollama Client (Only once!)
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🚀 Summarizer Worker Started...")

	// 3. THE INFINITE LOOP
	for {
		var id int
		var content string

		// Look for ONE un-summarized article
		query := "SELECT id, raw_content FROM scraped_pages WHERE is_summarized = false LIMIT 1"
		err = db.QueryRow(ctx, query).Scan(&id, &content)

		if err != nil {
			// If no rows are found, we don't 'break' anymore. 
			// We wait 5 seconds and try again.
			fmt.Println("😴 No new articles. Waiting 5 seconds...")
			time.Sleep(5 * time.Second)
			continue // Go back to the start of the 'for' loop
		}

		// 4. Process with AI
		fmt.Printf("🤖 Processing ID %d...\n", id)

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
			log.Printf("❌ AI Error for ID %d: %v", id, err)
			continue // Skip this one and try the next
		}

		// 5. Update the DB
		updateQuery := "UPDATE scraped_pages SET summary = $1, is_summarized = true WHERE id = $2"
		_, err = db.Exec(ctx, updateQuery, summaryText, id)
		if err != nil {
			log.Printf("❌ Update failed for ID %d: %v", id, err)
		} else {
			fmt.Printf("✅ Summary saved for ID %d\n", id)
		}

		// (Optional) Small pause so we don't melt the CPU
		time.Sleep(500 * time.Millisecond)
	}
}