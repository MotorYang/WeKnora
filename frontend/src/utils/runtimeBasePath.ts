/** Normalize a router/static/API base to one leading and one trailing slash. */
export function normalizeRuntimeBasePath(value: unknown): string {
  if (typeof value !== 'string') return '/'
  let path = value.trim()
  if (!path || path === '/' || path === './') return '/'
  if (!path.startsWith('/')) path = `/${path}`
  path = path.replace(/\/+$/, '')
  return path ? `${path}/` : '/'
}

/**
 * Docker writes BASE_PATH at container startup. Local development falls back
 * to Vite's build-time base so VITE_BASE_PATH continues to work there.
 */
export function getRuntimeBasePath(): string {
  const configured = typeof window === 'undefined'
    ? undefined
    : window.__RUNTIME_CONFIG__?.BASE_PATH
  return normalizeRuntimeBasePath(configured || import.meta.env?.BASE_URL || '/')
}

/** Base without a trailing slash, suitable for axios and URL concatenation. */
export function getRuntimeBasePrefix(): string {
  const base = getRuntimeBasePath()
  return base === '/' ? '' : base.slice(0, -1)
}
