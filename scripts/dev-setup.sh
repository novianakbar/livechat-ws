#!/bin/bash

# Development setup script for LiveChat WebSocket Server

set -e

echo "🚀 Setting up LiveChat WebSocket Server development environment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose is not installed. Please install docker-compose and try again."
    exit 1
fi

echo "✅ Docker is running"

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from .env.example..."
    cp .env.example .env
    echo "✅ .env file created. You can modify it if needed."
else
    echo "✅ .env file already exists"
fi

# Start infrastructure services
echo "🐳 Starting infrastructure services (Kafka, Redis)..."
docker-compose up -d zookeeper kafka redis

# Wait for Kafka to be ready
echo "⏳ Waiting for Kafka to be ready..."
sleep 30

# Create Kafka topics
echo "📋 Creating Kafka topics..."
docker-compose exec kafka kafka-topics --create --topic chat-messages --bootstrap-server localhost:29092 --partitions 3 --replication-factor 1 --if-not-exists
docker-compose exec kafka kafka-topics --create --topic typing-indicators --bootstrap-server localhost:29092 --partitions 3 --replication-factor 1 --if-not-exists
docker-compose exec kafka kafka-topics --create --topic connection-status --bootstrap-server localhost:29092 --partitions 3 --replication-factor 1 --if-not-exists

# List created topics
echo "📋 Created topics:"
docker-compose exec kafka kafka-topics --list --bootstrap-server localhost:29092

# Start monitoring services
echo "🖥️  Starting monitoring services..."
docker-compose up -d redis-commander kafka-ui

echo ""
echo "✅ Development environment is ready!"
echo ""
echo "📋 Available services:"
echo "   • Kafka: localhost:9092"
echo "   • Redis: localhost:6379"
echo "   • Kafka UI: http://localhost:8083"
echo "   • Redis Commander: http://localhost:8082"
echo ""
echo "🚀 To start the LiveChat WebSocket Server:"
echo "   • Local development: make run-env"
echo "   • Docker container: docker-compose up livechat-ws"
echo ""
echo "🧪 To test the setup:"
echo "   • Health check: curl http://localhost:8081/health"
echo "   • Connection status: curl http://localhost:8081/api/v1/session/123e4567-e89b-12d3-a456-426614174000/connection-status"
echo "   • WebSocket test: wscat -c ws://localhost:8081/ws/123e4567-e89b-12d3-a456-426614174000/user123/customer"
echo ""
echo "🛑 To stop all services: docker-compose down"
