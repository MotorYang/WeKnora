import assert from 'node:assert/strict'
import test from 'node:test'
import { rewriteEmbedDevUrl } from './vite.config'

test('embed dev fallback rewrites root deployments to embed.html', () => {
  assert.equal(rewriteEmbedDevUrl('/embed/channel-1', '/'), '/embed.html')
  assert.equal(rewriteEmbedDevUrl('/embed/channel-1?preview=1', '/'), '/embed.html?preview=1')
})

test('embed dev fallback preserves a configured Vite base exactly once', () => {
  assert.equal(
    rewriteEmbedDevUrl('/weknora/embed/channel-1', '/weknora/'),
    '/weknora/embed.html',
  )
})

test('embed dev fallback ignores other pages and asset-like paths', () => {
  assert.equal(rewriteEmbedDevUrl('/weknora/platform/chat', '/weknora/'), undefined)
  assert.equal(rewriteEmbedDevUrl('/weknora/embed/channel.js', '/weknora/'), undefined)
})
