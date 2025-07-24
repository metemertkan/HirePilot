# Build stage
FROM node:20-alpine AS build
WORKDIR /app

COPY frontend ./frontend
WORKDIR /app/frontend

RUN npm install
RUN npm run build

# Production stage
FROM nginx:alpine
COPY --from=build /app/frontend/build /usr/share/nginx/html
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]