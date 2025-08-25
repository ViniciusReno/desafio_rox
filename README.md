# Desafio Cotação B3

## Setup

1. Copy `.env.example` to `.env` and adjust the values as needed:

   ```sh
   cp .env.example .env
   ```

2. Build the binaries:

```sh
make build
```

3. Start PostgreSQL and the API:

```sh
docker-compose up -d
```

## Data Ingestion

The ingestion service checks the previous business day and imports data if it hasn't been processed yet. Run the service with:

```sh
make ingest
```

Leave it running to process new data daily.

### Offline Ingestion

To ingest data without network access:

1. Create a `data/` directory in the project root.
2. Manually download the daily ZIP files from B3's ticker CSV endpoint. Each archive must contain a CSV with the header:

   ```text
   DT_NEG;TICKER;PRECO;QUANTIDADE;HORA
   ```

   Lines are semicolon-separated and `PRECO` uses a comma as the decimal separator (e.g., `10,5`).
3. Rename each downloaded file to the corresponding date (e.g., `2024-05-05`) and place it in the `data/` directory.
4. Run the ingestion service pointing to this directory by overriding the base URL:

   ```sh
   make ingest ARGS='-ldflags "-X main.b3BaseURL=file://$(pwd)/data"'
   ```

   You can alternatively provide the path via a flag or environment variable that the `ingest` binary understands.

This process reads the local CSVs and ingests them into the database without downloading new files.

## Running the API

If the containers are not already running, start them:

```sh
docker-compose up -d
```

The API will be available on `http://localhost:8080`.

To run the API directly without Docker:

```sh
make run
```

## Example Request

```sh
curl "http://localhost:8080/quotes/summary?ticker=TEST&date_start=2024-05-01"
```

## Tests

Run the unit tests with:

```sh
make test
```

