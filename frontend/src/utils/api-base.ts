import { getRuntimeBasePrefix } from './runtimeBasePath'

export function getApiBaseUrl(): string {
  return getRuntimeBasePrefix()
}
