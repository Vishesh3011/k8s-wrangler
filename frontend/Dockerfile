FROM nginx:alpine

WORKDIR /usr/share/nginx/html

# Copy frontend sources (index.html, app.js, css, etc.)
COPY . .

# Provide nginx config and an entrypoint to generate runtime config.js
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]