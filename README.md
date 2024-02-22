# Simple mongoDB server with GO

## Setup

1. Clone the repository to your local machine.
    ```cmd
    git clone https://github.com/your-username/go-server-mahangoDB.git
    ```

2. Navigate to the project directory.
    ```cmd
    cd go-server-mahangoDB
    ```

## Environment Variables

1. Create a new file named `.env` in the project root directory.
    ```cmd
    type nul > .env
    ```

2. Open the `.env` file using a text editor, and add the following environment variables:
    ```
    MONGODB_CONNECTION_STRING=<your-mongodb-connection-string>
    ```
    Replace <your-mongodb-connection-string> with the connection string obtained from your MongoDB Atlas dashboard.

## Running the Server

1. Download Go:
   Visit the official [Go download page](https://golang.org/dl/) and install Go as per the documentation.

2. Install Dependencies:
    ```cmd
    go mod tidy
    ```

3. Run the Server:
    ```cmd
    go run main.go
    ```
    
4. Access the API Endpoint:
   Once the server is running, you can access the API endpoints by making HTTP requests. By default, the server will run on localhost:8080.
   - To fetch user data, make a GET request to `http://localhost:8080/api/ocular?userName=<username>`.
   - To post user data, make a POST request to `http://localhost:8080/api/ocular` with JSON data containing user information.

   Note: the username is to registered in the mongoDB.
