FROM node:22-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --omit=dev --ignore-scripts && npm cache clean --force

COPY index.js db.js ./

ENV NODE_ENV=production \
	APP_PORT=8080 \
	NODE_OPTIONS=--max-old-space-size=96

USER node

EXPOSE 8080

CMD ["node", "index.js"]
