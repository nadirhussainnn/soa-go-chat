# Use Go base image
FROM golang:1.20

# Set the working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the source code
COPY . .

# Expose the port used by the service
EXPOSE 8080

# Set environment variables (optional, can use Compose for this too)
ENV PORT=8080
ENV AUTH_SERVICE_URL=http://auth-service:8081
ENV MESSAGING_SERVICE_URL=http://messaging-service:8082
ENV CONTACTS_SERVICE_URL=http://contacts-service:8083
ENV FRONTEND_URL=http://consumer:8085
ENV CONTACTS_SERVICE_WS_URL=ws://contacts-service:8083/ws
ENV MESSAGING_SERVICE_WS_URL=ws://messaging-service:8082/ws

# Build the application
RUN go build -o main .

# Run the application
CMD ["./main"]
