# Use an official Golang runtime as a parent image
FROM golang:latest

# Set the working directory to /go/src/app
WORKDIR /go/src/app

# Copy the current directory contents into the container at /go/src/app
COPY . .

# Download and install any required third-party dependencies into the container
RUN go get -d -v ./...

# Build the Go application
RUN go build -o main

# Expose port 8002 to the outside world
EXPOSE 8083

# Command to run the executable
CMD ["./main"]
