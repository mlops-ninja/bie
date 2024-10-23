# Build stage
FROM rust:1.82-slim-bullseye as builder

WORKDIR /usr/src/bie-server

# Copy the entire project
COPY . .

# Build the project
RUN cargo build --release --bin bie-server

# Runtime stage
FROM gcr.io/distroless/cc-debian12

WORKDIR /usr/local/bin

# Copy the built binary from the builder stage
COPY --from=builder /usr/src/bie-server/target/release/bie-server .

# Set environment variables with default values
ENV BIE_PORT=3000
ENV BIE_MAX_FILE_SIZE=10485760

# Expose the port
EXPOSE ${BIE_PORT}

# Run the binary
CMD ["./bie-server"]
