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
EXPOSE 8083

# Set environment variables (optional, can use Compose for this too)
ENV PORT=8083
ENV DB_NAME=contacts.db
ENV GATEWAY_URL=http://gateway:8080
ENV AMQP_URL=amqp://guest:guest@rabbitmq:5672/
# AMQP_URL="amqps://gckokhqp:2FUs3N9wEqJj7a1GF3FD4dabAzoA1ezq@kangaroo.rmq.cloudamqp.com/gckokhqp"

# Build the application
RUN go build -o main .

# Run the application
CMD ["./main"]
