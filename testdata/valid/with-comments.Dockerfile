# This Dockerfile has extensive comments
# Author: Test Developer
# Purpose: Demonstrate comment handling

# Base image selection
# Using Alpine for minimal footprint
FROM alpine:3.18

# Install required packages
# Note: We clean up cache to reduce image size
RUN apk add --no-cache curl

# Set up application directory
# This will be the working directory for all subsequent commands
WORKDIR /app

# Copy application files
# The source files should be in the build context
COPY . .

# Configure the user
# Running as non-root for security
USER nobody

# Health check configuration
# Checks every 30 seconds with 10 second timeout
HEALTHCHECK --interval=30s --timeout=10s CMD curl -f http://localhost/ || exit 1

# Default command
# Can be overridden at runtime
CMD ["./start.sh"]
