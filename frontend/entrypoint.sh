#!/bin/sh

# Default backend if not provided
: "${BACKEND_URL:=http://localhost:8080}"

cat > /usr/share/nginx/html/config.js <<EOF
window.API_BASE_URL = "${BACKEND_URL}";
EOF

# Execute the CMD
exec "$@"
