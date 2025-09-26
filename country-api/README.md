# Golang Country Search API

This is a REST API service built in Go that provides country information by leveraging the [REST Countries API](https://restcountries.com/). It includes custom in-memory caching and graceful shutdown.

## Core Features

- Single endpoint to search for a country by name.
- Thread-safe, in-memory caching to reduce latency and API calls.
- Proper error handling and logging.
- Graceful shutdown to handle in-flight requests.
- High unit test coverage (>90%).

## Prerequisites

- Go 1.21 or later.

## Setup & Running

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/your-username/country-api.git
    cd country-api
    ```

2.  **Install dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Run the server:**
    ```bash
    go run ./cmd/server/main.go
    ```
    The server will start on `http://localhost:8000`.

## API Documentation

### Search for a Country

- **Endpoint:** `GET /api/countries/search`
- **Query Parameter:** `name` (string, required) - The name of the country to search for.
- **Success Response (200 OK):**
  ```json
  {
    "name": "India",
    "capital": "New Delhi",
    "currency": "₹",
    "population": 1380004385
  }
  ```
- **Error Responses:**
  - `400 Bad Request`: If the `name` query parameter is missing.
  - `404 Not Found`: If the country cannot be found.
  - `500 Internal Server Error`: For any other server-side errors.

#### Example Request

```bash
curl -X GET "http://localhost:8000/api/countries/search?name=Germany"
```
