import assert from 'node:assert/strict'
import test from 'node:test'
import {
  getRuntimeBasePath,
  getRuntimeBasePrefix,
  normalizeRuntimeBasePath,
} from './runtimeBasePath'

test('normalizeRuntimeBasePath produces a canonical router base', () => {
  assert.equal(normalizeRuntimeBasePath(undefined), '/')
  assert.equal(normalizeRuntimeBasePath(''), '/')
  assert.equal(normalizeRuntimeBasePath('/'), '/')
  assert.equal(normalizeRuntimeBasePath('./'), '/')
  assert.equal(normalizeRuntimeBasePath('weknora'), '/weknora/')
  assert.equal(normalizeRuntimeBasePath('/weknora/'), '/weknora/')
  assert.equal(normalizeRuntimeBasePath('/apps/weknora///'), '/apps/weknora/')
})

test('runtime BASE_PATH controls router and API prefixes', () => {
  const previousWindow = globalThis.window
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: { __RUNTIME_CONFIG__: { BASE_PATH: '/weknora/' } },
  })

  try {
    assert.equal(getRuntimeBasePath(), '/weknora/')
    assert.equal(getRuntimeBasePrefix(), '/weknora')
  } finally {
    if (previousWindow === undefined) {
      Reflect.deleteProperty(globalThis, 'window')
    } else {
      Object.defineProperty(globalThis, 'window', {
        configurable: true,
        value: previousWindow,
      })
    }
  }
})
