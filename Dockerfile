FROM python:3-slim-bookworm

# Set working directory
WORKDIR /app

# Install system dependencies for serial communication
RUN apt-get update && apt-get install -y \
    udev \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create non-root user for security
RUN useradd -m -u 1000 p1user && chown -R p1user:p1user /app

# Switch to non-root user
USER p1user

# Expose port
EXPOSE 9039

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9039/latest || exit 1

# Run the application
CMD ["python3", "main.py"]