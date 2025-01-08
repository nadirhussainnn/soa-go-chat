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
EXPOSE 8081

# Set environment variables (optional, can use Compose for this too)
ENV PORT=8081
ENV JWT_SECRET=6da7s86d78##$3s4390d78as&*67dsa5675
ENV DB_NAME=auth.db
ENV AMQP_URL=amqp://guest:guest@rabbitmq:5672/

# Build the application
RUN go build -o main .

# Run the application
CMD ["./main"]
