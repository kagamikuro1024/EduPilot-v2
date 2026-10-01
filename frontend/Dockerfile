# Ngữ cảnh build = gốc repo (lockfile pnpm nằm ở gốc workspace): docker build -f frontend/Dockerfile .
FROM node:24-alpine AS build
RUN npm install -g pnpm@12.8.1
WORKDIR /app
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY frontend/package.json frontend/
RUN pnpm install --frozen-lockfile --filter frontend
COPY frontend frontend
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm -C frontend build

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
COPY --from=build --chown=node:node /app/frontend/.next/standalone ./
COPY --from=build --chown=node:node /app/frontend/.next/static ./frontend/.next/static
COPY --from=build --chown=node:node /app/frontend/public ./frontend/public
USER node
EXPOSE 3000
CMD ["node", "frontend/server.js"]
