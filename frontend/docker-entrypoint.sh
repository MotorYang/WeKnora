#!/bin/sh

set -eu

# A single image can run at / or below a reverse-proxy path such as /weknora/.
# Empty and / disable the prefix. Keep the accepted alphabet deliberately
# narrow because this value is written into JavaScript, HTML, and nginx config.
RAW_FRONTEND_BASE_PATH=${FRONTEND_BASE_PATH:-/}
case "$RAW_FRONTEND_BASE_PATH" in
  ""|/)
    RUNTIME_BASE_PATH=/
    FRONTEND_PATH_PREFIX=""
    ;;
  *)
    case "$RAW_FRONTEND_BASE_PATH" in
      /*) FRONTEND_PATH_PREFIX=$RAW_FRONTEND_BASE_PATH ;;
      *) FRONTEND_PATH_PREFIX=/$RAW_FRONTEND_BASE_PATH ;;
    esac
    FRONTEND_PATH_PREFIX=$(printf '%s' "$FRONTEND_PATH_PREFIX" | sed 's:/*$::')
    if ! printf '%s' "$FRONTEND_PATH_PREFIX" | grep -Eq '^(/[A-Za-z0-9_-]+)+$'; then
      echo "Invalid FRONTEND_BASE_PATH: use path segments containing only letters, digits, _ or -" >&2
      exit 1
    fi
    RUNTIME_BASE_PATH=${FRONTEND_PATH_PREFIX}/
    ;;
esac

# Only emit whitelisted locale tags to avoid config.js injection from env values.
RUNTIME_DEFAULT_LOCALE=""
case "${DEFAULT_LOCALE:-}" in
  zh-CN|zh-TW|en-US|ru-RU|ko-KR|ja-JP) RUNTIME_DEFAULT_LOCALE="${DEFAULT_LOCALE}" ;;
esac

# 生成运行时配置文件，注入环境变量到前端
FILE_MB=${MAX_FILE_SIZE_MB:-50}
SKILL_MB=${MAX_SKILL_BUNDLE_SIZE_MB:-256}
if [ "$SKILL_MB" -lt "$FILE_MB" ] 2>/dev/null; then
  SKILL_MB=$FILE_MB
fi
if [ "$SKILL_MB" -gt 512 ] 2>/dev/null; then
  SKILL_MB=512
fi

cat > /usr/share/nginx/html/config.js << EOF
window.__RUNTIME_CONFIG__ = {
  MAX_FILE_SIZE_MB: ${FILE_MB},
  MAX_SKILL_BUNDLE_SIZE_MB: ${SKILL_MB},
  DEFAULT_LOCALE: "${RUNTIME_DEFAULT_LOCALE}",
  BASE_PATH: "${RUNTIME_BASE_PATH}"
};
EOF

# The image is built with relative asset URLs. An absolute runtime <base>
# keeps those URLs stable on deep SPA routes such as /weknora/platform/chat.
for HTML_FILE in /usr/share/nginx/html/index.html /usr/share/nginx/html/embed.html; do
  if [ ! -f "$HTML_FILE" ]; then
    continue
  fi
  if grep -q 'id="weknora-runtime-base"' "$HTML_FILE"; then
    sed -i "s|<base id=\"weknora-runtime-base\" href=\"[^\"]*\">|<base id=\"weknora-runtime-base\" href=\"${RUNTIME_BASE_PATH}\">|" "$HTML_FILE"
  else
    sed -i "s|<head>|<head>\n    <base id=\"weknora-runtime-base\" href=\"${RUNTIME_BASE_PATH}\">|" "$HTML_FILE"
  fi
done

# Prefix requests are internally re-run through the existing root locations.
# This preserves all current API, files, MCP, embed, WebSocket, and SPA rules.
: > /etc/nginx/path-prefix.conf
if [ -n "$FRONTEND_PATH_PREFIX" ]; then
  cat > /etc/nginx/path-prefix.conf << EOF
location = ${FRONTEND_PATH_PREFIX} {
    absolute_redirect off;
    return 301 ${RUNTIME_BASE_PATH};
}

location ^~ ${RUNTIME_BASE_PATH} {
    rewrite ^${FRONTEND_PATH_PREFIX}(/.*)\$ \$1 last;
}
EOF
fi

# 处理 nginx 配置。
# 两个上限分开注入：全站保持知识库的 MAX_FILE_SIZE，只有技能 zip 上传的两条
# 集合路由放宽到 MAX_SKILL_BUNDLE_SIZE（不含 /install、PATCH 等子路径）。
# 合成一个全站上限会让每个上传端点都能收到技能包那么大的 body。
# client_max_body_size 计的是整个请求体（文件 + multipart 分隔符/文件名等），
# 前端按文件本身大小校验。上限若恰等于 FILE_MB，正好 FILE_MB 的文件会被
# nginx 以 413 拒绝，所以给 multipart 开销留 1M 余量。
export MAX_FILE_SIZE=$((FILE_MB + 1))M
export MAX_SKILL_BUNDLE_SIZE=${SKILL_MB}M
export APP_HOST=${APP_HOST:-app}
export APP_PORT=${APP_PORT:-8080}
export APP_SCHEME=${APP_SCHEME:-http}
envsubst '${MAX_FILE_SIZE} ${MAX_SKILL_BUNDLE_SIZE} ${APP_HOST} ${APP_PORT} ${APP_SCHEME}' \
  < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf

# 启动 nginx
exec nginx -g 'daemon off;'
