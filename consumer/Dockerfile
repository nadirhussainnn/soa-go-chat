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
EXPOSE 8085

# Set environment variables (optional, can use Compose for this too)
ENV PORT=8085
ENV GATEWAY_URL=http://gateway:8080
ENV AMQP_URL=amqp://guest:guest@rabbitmq:5672/
ENV GATEWAY_WS_URL=ws://gateway:8080/ws

# Build the application
RUN go build -o main .

# Run the application
CMD ["./main"]
