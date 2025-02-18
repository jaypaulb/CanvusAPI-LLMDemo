#!/bin/bash
echo "Pulling latest changes..."
git pull

echo "Stopping apiDemo..."
pm2 stop apiDemo

echo "Building go_backend..."
go build -o go_backend

echo "Starting apiDemo..."
pm2 start apiDemo

echo "Update complete!" 