FROM golang:latest

# Enable Go Modules
ENV GO111MODULE=on

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/src/github.com/furkansenharputlu/f-license

# Copy everything from the current directory to the PWD (Present Working Directory) inside the container
COPY . .

# Build
RUN go build -mod=vendor

# This container exposes port 4242 to the outside world
EXPOSE 4242

# Run the executable
CMD ["./f-license"]