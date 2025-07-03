# Backend Job Listing API

This project is a simple HTTP server that handles job listings. It allows users to add job postings and retrieve a list of all job postings.

## Files

- **main.go**: Contains the main application logic for the HTTP server, including the `Job` struct and handlers for adding and listing jobs.
- **Dockerfile**: Used to build a Docker image for the Go application.

## Setup Instructions

1. **Clone the repository**:
   ```
   git clone <repository-url>
   cd Backend
   ```

2. **Build the Docker image**:
   ```
   docker build -t job-listing-api .
   ```

3. **Run the Docker container**:
   ```
   docker run -p 8080:8080 job-listing-api
   ```

## Usage

- **Add a Job**:
  - Endpoint: `POST /api/jobs`
  - Request Body:
    ```json
    {
      "title": "Job Title",
      "company": "Company Name",
      "link": "http://example.com/job"
    }
    ```

- **List Jobs**:
  - Endpoint: `GET /api/jobs`
  - Response:
    ```json
    [
      {
        "title": "Job Title",
        "company": "Company Name",
        "link": "http://example.com/job"
      }
    ]
    ```

## License

This project is licensed under the MIT License.