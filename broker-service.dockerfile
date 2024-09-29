# Start with a base image (Alpine is lightweight)
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the pre-built binary into the container
COPY brokerApp /app/brokerApp

# Expose the necessary port
EXPOSE 80

# Run the binary
CMD ["./brokerApp"]
