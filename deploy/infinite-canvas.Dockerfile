FROM alpine/git:2.49.1 AS source

ARG INFINITE_CANVAS_COMMIT=b66936d891b82c2b51c1ed05e1a6eae3e31d4ca3
WORKDIR /src
RUN git clone https://github.com/basketikun/infinite-canvas.git . && git checkout "${INFINITE_CANVAS_COMMIT}"
COPY infinite-canvas-base.patch /tmp/infinite-canvas-base.patch
RUN git apply /tmp/infinite-canvas-base.patch

FROM oven/bun:1.3.13 AS build

WORKDIR /app/web
COPY --from=source /src/web/package.json /src/web/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile --cache-dir=/root/.bun/install/cache
COPY --from=source /src/VERSION /app/VERSION
COPY --from=source /src/CHANGELOG.md /app/CHANGELOG.md
COPY --from=source /src/web ./
ENV VITE_BASE=/canvas-app/
RUN bun run build

FROM nginx:1.27-alpine

COPY --from=build /app/web/dist /usr/share/nginx/html
COPY --from=source /src/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=source /src/web/docker-entrypoint.sh /docker-entrypoint.d/40-runtime-config.sh
RUN chmod +x /docker-entrypoint.d/40-runtime-config.sh

EXPOSE 3000
