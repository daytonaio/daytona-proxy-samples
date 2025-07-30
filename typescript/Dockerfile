FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY . .

# Build the TypeScript code
RUN npm run build

ENV PORT=1234
ENV DAYTONA_API_URL=https://app.daytona.io/api

# Expose the port
EXPOSE ${PORT}

# Start the application
CMD ["node", "dist/index.js"]
