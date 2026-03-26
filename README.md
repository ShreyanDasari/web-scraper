# Getting Started

## 📋 Prerequisites
* Docker & Docker Compose installed.

* Ollama installed locally (if not running inside Docker).

* Go 1.22+ (if you wish to run services outside of containers).

##  🛠️ 1. Setup Environment

Clone the repository and create your .env file from the template:

```bash
git clone https://github.com/ShreyanDasari/web-scraper.git
cd web-scraper
```
Create the .env file
```
touch .env

# Open it in your editor (or use 'nano .env')
# Add these exact lines:
DB_URL=postgres://user:password@db:5432/scraper_db
REDIS_URL=redis:6379
OLLAMA_HOST=http://host.docker.internal:11434
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=scraper_db
```
## 📦 2. Launch the Infrastructure
Use Docker Compose to build and start the PostgreSQL database, Redis broker, and the Go services:

```Bash
docker-compose up --build -d
```
The -d flag runs the containers in the background, keeping your terminal clean.

### 🧠 3. Prepare the AI Model
Ensure Ollama is running and pull the lightweight smollm2 model used by the Summarizer Service:

```Bash
docker exec -it web-scraper-ollama-1 ollama pull smollm2
```
### 📈 4. Scaling the Workers (Optional)
To demonstrate the Competing Consumers pattern and speed up processing for large batches, you can scale the summarizer workers horizontally:

```Bash
docker-compose up -d --scale summarizer=3
```
### 🔍 5. Verifying the Data
You can check the logs to see the services communicating in real-time, or query the database directly:

View Logs:
```Bash
docker-compose logs -f summarizer
```
Check Database (via Docker):
```Bash
docker exec -it scraper-db psql -U user -d scraper_db -c "SELECT title, summary FROM scraped_pages WHERE is_summarized = true;"
```

# Architecture & Workflow:-

This project implements an Asynchronous, Event-Driven Architecture to decouple data ingestion from intensive AI processing. By using Redis as a message broker, the system achieves horizontal scalability and high fault tolerance.

## The Data Lifecycle:-

**Ingestion:** The Scraper Service (Go + Colly) fetches raw HTML, parses the content, and performs an atomic INSERT into Postgres.

**Handshake:** Using the RETURNING id clause, the Scraper retrieves the unique primary key and immediately LPushes it to the Redis scrape_tasks list.

**Messaging:** The Summarizer Service remains idle in a Blocking Pop (BRPop) state, consuming 0% CPU until a task ID arrives in the queue.

**AI Processing:** Once a worker pulls an ID, it fetches the raw content from Postgres and sends it to the Ollama API (running smollm2) for summarization.

**Persistence:** The final summary is saved back to the database, and the record is marked as is_summarized = true.
#
<p align="center">
<img width="741" height="392" alt="Untitled Diagram drawio (1)" src="https://github.com/user-attachments/assets/b8469b93-7c00-45ec-9676-959e656c1ec0" />
</p>

## Key Technical Features
* Decoupled Services: Scraping and AI processing run in independent Docker containers, preventing a slow AI response from bottlenecking the scraper.

* Competing Consumers: Multiple Summarizer workers can be spun up (Auto-scaled) to process the Redis queue in parallel without task duplication.

* Atomic Data Integrity: Postgres UNIQUE constraints and ON CONFLICT handling prevent redundant scraping and data corruption.

