#!/bin/bash

# Script to build and run the integracaocron application

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Building IntegracaoCron Application ===${NC}"

# Build the application
echo -e "${YELLOW}Building Go application...${NC}"
go build -o bin/integracaocron ./cmd/app/

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Build successful!${NC}"
else
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found. Creating template...${NC}"
    cat > .env << EOF
# Database Configuration
DB_DIALECT=oracle
DB_USER=your_db_user
DB_PASSWD=your_db_password
DB_SCHEMA=your_schema
DB_CONNECTSTRING=your_connection_string

# RabbitMQ Configuration
ENV_RABBITMQ=amqp://user:password@localhost:5672/

# Redis Configuration (if needed)
ENV_REDIS_ADDRESS=localhost:6379
ENV_REDIS_PASSWORD=
ENV_REDIS_EXPIRE=3600

# Application Configuration
WORKERS=20
EOF
    echo -e "${RED}Please configure the .env file with your database and RabbitMQ settings${NC}"
    exit 1
fi

# Run the application
echo -e "${GREEN}Starting IntegracaoCron...${NC}"
./bin/integracaocron