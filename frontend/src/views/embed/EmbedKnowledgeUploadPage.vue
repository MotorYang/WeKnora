<template>
  <main class="embed-knowledge" :style="pageStyle">
    <header class="embed-knowledge__header">
      <div>
        <h1>{{ t('embedKnowledge.manageTitle') }}</h1>
        <!--<p>{{ t('embedKnowledge.manageDescription') }}</p>-->
      </div>
      <t-button tag="a" :href="chatHref" variant="outline">
        <template #icon><t-icon name="chat" /></template>
        {{ t('embedKnowledge.backToChat') }}
      </t-button>
    </header>

    <section v-if="loadError" class="embed-knowledge__state">
      <t-alert theme="error" :message="loadError" />
    </section>
    <section v-else-if="bootstrapping || !config" class="embed-knowledge__state">
      <t-loading /><span>{{ t('embedPublish.loading') }}</span>
    </section>
    <section v-else-if="config.allow_file_upload !== true" class="embed-knowledge__state">
      <t-alert theme="warning" :message="t('embedKnowledge.disabled')" />
    </section>
    <section v-else class="embed-knowledge__workspace">
      <div class="embed-knowledge__toolbar">
        <t-select v-model="selectedKbId" class="embed-knowledge__kb-select" :options="knowledgeBaseOptions"
          :loading="knowledgeBasesLoading" :placeholder="t('embedKnowledge.destinationPlaceholder')" />
        <t-input v-model="keywordInput" class="embed-knowledge__search" clearable
          :placeholder="t('embedKnowledge.searchPlaceholder')" @enter="applySearch" @clear="applySearch">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-button variant="outline" :loading="activeLoading" @click="refreshActive">
          <template #icon><t-icon name="refresh" /></template>{{ t('embedKnowledge.refresh') }}
        </t-button>
        <t-button theme="primary" @click="uploadPanelVisible = !uploadPanelVisible">
          <template #icon><t-icon name="upload" /></template>{{ t('embedKnowledge.uploadDocument') }}
        </t-button>
      </div>

      <t-alert v-if="knowledgeBasesError" theme="error" :message="knowledgeBasesError" />

      <section v-if="uploadPanelVisible" class="embed-knowledge__upload-panel">
        <label class="embed-knowledge__dropzone" :class="{ 'is-dragging': dragging }"
          @dragenter.prevent="dragging = true" @dragover.prevent="dragging = true"
          @dragleave.prevent="dragging = false" @drop.prevent="handleDrop">
          <input type="file" multiple :accept="acceptedFileTypes" @change="handleFileInput" />
          <t-icon name="upload" size="30px" />
          <strong>{{ t('embedKnowledge.chooseFiles') }}</strong>
          <span>{{ t('embedKnowledge.dropHint') }}</span>
        </label>
        <ul v-if="uploads.length" class="embed-knowledge__upload-list">
          <li v-for="entry in uploads" :key="entry.key">
            <t-icon name="file" />
            <div class="embed-knowledge__file-main">
              <span :title="entry.file.name">{{ entry.file.name }}</span>
              <small>{{ formatFileSize(entry.file.size) }}</small>
              <t-progress v-if="entry.status === 'uploading'" :percentage="entry.progress" size="small" />
              <small v-else-if="entry.status === 'success'" class="is-success">{{ t('embedKnowledge.uploadSuccess') }}</small>
              <small v-else-if="entry.status === 'error'" class="is-error">{{ entry.error || t('embedKnowledge.uploadFailed') }}</small>
            </div>
            <t-button v-if="entry.status === 'pending'" variant="text" shape="square" @click="removeUpload(entry.key)">
              <template #icon><t-icon name="close" /></template>
            </t-button>
          </li>
        </ul>
        <div class="embed-knowledge__upload-actions">
          <span>{{ t('embedKnowledge.securityHint') }}</span>
          <t-button theme="primary" :loading="uploading"
            :disabled="!selectedKbId || !uploads.some(item => item.status === 'pending')" @click="uploadAll">
            {{ t('embedKnowledge.upload') }}
          </t-button>
        </div>
      </section>

      <t-tabs v-model="activeTab" class="embed-knowledge__tabs" @change="onTabChange">
        <t-tab-panel value="documents" :label="t('embedKnowledge.documents')" />
        <t-tab-panel value="gallery" :label="t('embedKnowledge.gallery')" />
      </t-tabs>

      <section v-if="activeTab === 'documents'" class="embed-knowledge__content">
        <div v-if="documentsLoading" class="embed-knowledge__loading"><t-loading /></div>
        <t-alert v-else-if="documentsError" theme="error" :message="documentsError" />
        <div v-else-if="documents.length === 0" class="embed-knowledge__empty">
          <t-icon name="file" size="42px" /><span>{{ t('embedKnowledge.noDocuments') }}</span>
        </div>
        <div v-else class="embed-knowledge__table-wrap">
          <table class="embed-knowledge__table">
            <thead><tr>
              <th>{{ t('embedKnowledge.fileName') }}</th><th>{{ t('embedKnowledge.status') }}</th>
              <th>{{ t('embedKnowledge.size') }}</th><th>{{ t('embedKnowledge.updatedAt') }}</th>
              <th>{{ t('embedKnowledge.actions') }}</th>
            </tr></thead>
            <tbody>
              <tr v-for="doc in documents" :key="doc.id">
                <td><div class="embed-knowledge__doc-name"><t-icon name="file" /><div>
                  <strong :title="documentName(doc)">{{ documentName(doc) }}</strong>
                  <small v-if="doc.description" :title="doc.description">{{ doc.description }}</small>
                </div></div></td>
                <td><t-tag size="small" :theme="statusTheme(doc.parse_status)">{{ statusLabel(doc.parse_status) }}</t-tag></td>
                <td>{{ doc.file_size ? formatFileSize(doc.file_size) : '—' }}</td>
                <td>{{ formatTime(doc.updated_at || doc.created_at) }}</td>
                <td><div class="embed-knowledge__row-actions">
                  <t-button variant="text" size="small" @click="previewDocument(doc)">{{ t('embedKnowledge.preview') }}</t-button>
                  <t-button variant="text" size="small" @click="downloadDocument(doc)">{{ t('embedKnowledge.download') }}</t-button>
                  <t-button variant="text" size="small" @click="openEdit(doc)">{{ t('embedKnowledge.edit') }}</t-button>
                  <t-popconfirm :content="t('embedKnowledge.deleteConfirm', { name: documentName(doc) })"
                    :confirm-btn="{ content: t('embedKnowledge.delete'), theme: 'danger' }" @confirm="deleteDocument(doc)">
                    <t-button theme="danger" variant="text" size="small">{{ t('embedKnowledge.delete') }}</t-button>
                  </t-popconfirm>
                </div></td>
              </tr>
            </tbody>
          </table>
        </div>
        <t-pagination v-if="documentsTotal > pageSize" v-model="documentsPage" :total="documentsTotal"
          :page-size="pageSize" :show-page-size="false" @change="loadDocuments" />
      </section>

      <section v-else class="embed-knowledge__content">
        <div v-if="galleryLoading" class="embed-knowledge__loading"><t-loading /></div>
        <t-alert v-else-if="galleryError" theme="error" :message="galleryError" />
        <div v-else-if="galleryImages.length === 0" class="embed-knowledge__empty">
          <t-icon name="image" size="42px" /><span>{{ t('embedKnowledge.noImages') }}</span>
        </div>
        <div v-else class="embed-knowledge__gallery">
          <button v-for="image in galleryImages" :key="image.id" type="button" @click="openImage(image)">
            <img v-if="galleryUrls[image.id]" :src="galleryUrls[image.id]" :alt="image.caption || image.source_name || ''" />
            <div v-else class="embed-knowledge__image-loading"><t-loading size="small" /></div>
            <span>{{ image.caption || image.source_name || t('embedKnowledge.untitledImage') }}</span>
          </button>
        </div>
        <t-pagination v-if="galleryTotal > galleryPageSize" v-model="galleryPage" :total="galleryTotal"
          :page-size="galleryPageSize" :show-page-size="false" @change="loadGallery" />
      </section>
    </section>

    <t-dialog v-model:visible="editVisible" :header="t('embedKnowledge.editDocument')"
      :confirm-btn="{ content: t('embedKnowledge.save'), loading: editSaving }" @confirm="saveEdit">
      <t-form label-align="top">
        <t-form-item :label="t('embedKnowledge.fileName')"><t-input v-model="editForm.title" :maxlength="500" /></t-form-item>
        <t-form-item :label="t('embedKnowledge.descriptionLabel')">
          <t-textarea v-model="editForm.description" :maxlength="2000" :autosize="{ minRows: 4, maxRows: 8 }" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="imageViewerVisible" width="min(920px, 92vw)" :footer="false">
      <img v-if="activeImageUrl" class="embed-knowledge__viewer-image" :src="activeImageUrl" :alt="activeImage?.caption || ''" />
      <h3>{{ activeImage?.caption || activeImage?.source_name }}</h3>
      <p v-if="activeImage?.ocr_text">{{ activeImage.ocr_text }}</p>
    </t-dialog>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  deleteEmbedKnowledgeDocument, getEmbedImageBlob, getEmbedKnowledgeBlob, listEmbedGalleryImages,
  listEmbedKnowledgeBases, listEmbedKnowledgeDocuments, updateEmbedKnowledgeDocument, uploadEmbedKnowledgeFile,
  type EmbedGalleryImage, type EmbedKnowledgeBaseOption, type EmbedKnowledgeDocument,
} from '@/api/embed'
import { useEmbedBridge } from '@/composables/useEmbedBridge'
import { getRuntimeBasePath } from '@/utils/runtimeBasePath'

type UploadStatus = 'pending' | 'uploading' | 'success' | 'error'
interface UploadEntry { key: string; file: File; status: UploadStatus; progress: number; error: string }

const acceptedFileTypes = '.pdf,.doc,.docx,.txt,.md,.csv,.xlsx,.xls,.ppt,.pptx,.epub,.mhtml,.xmind,.json,.html,.htm'
const route = useRoute()
const { t, locale } = useI18n()
const channelId = ref(String(route.params.channelId || ''))
const { token, config, loadError, bootstrapping } = useEmbedBridge(channelId)
const knowledgeBases = ref<EmbedKnowledgeBaseOption[]>([])
const selectedKbId = ref('')
const knowledgeBasesLoading = ref(false)
const knowledgeBasesError = ref('')
const uploads = ref<UploadEntry[]>([])
const uploading = ref(false)
const dragging = ref(false)
const uploadPanelVisible = ref(false)
const activeTab = ref<'documents' | 'gallery'>('documents')
const keywordInput = ref('')
const keyword = ref('')
const documents = ref<EmbedKnowledgeDocument[]>([])
const documentsLoading = ref(false)
const documentsError = ref('')
const documentsPage = ref(1)
const documentsTotal = ref(0)
const pageSize = 20
const galleryImages = ref<EmbedGalleryImage[]>([])
const galleryLoading = ref(false)
const galleryError = ref('')
const galleryPage = ref(1)
const galleryTotal = ref(0)
const galleryPageSize = 24
const galleryUrls = ref<Record<string, string>>({})
const objectUrls = new Set<string>()
const editVisible = ref(false)
const editSaving = ref(false)
const editingDocument = ref<EmbedKnowledgeDocument | null>(null)
const editForm = reactive({ title: '', description: '' })
const imageViewerVisible = ref(false)
const activeImage = ref<EmbedGalleryImage | null>(null)

const pageStyle = computed(() => config.value?.primary_color
  ? { '--td-brand-color': config.value.primary_color, '--embed-primary': config.value.primary_color } : {})
const knowledgeBaseOptions = computed(() => knowledgeBases.value.map(kb => ({ label: kb.name || kb.id, value: kb.id })))
const activeLoading = computed(() => activeTab.value === 'documents' ? documentsLoading.value : galleryLoading.value)
const chatHref = computed(() => {
  const path = `${getRuntimeBasePath()}embed/${encodeURIComponent(channelId.value)}`
  return token.value ? `${path}#token=${encodeURIComponent(token.value)}` : path
})
const activeImageUrl = computed(() => activeImage.value ? galleryUrls.value[activeImage.value.id] : '')

watch([token, config], async ([apiToken, channelConfig]) => {
  if (apiToken && channelConfig?.allow_file_upload === true) await loadKnowledgeBases()
}, { immediate: true })

watch(selectedKbId, async (next, previous) => {
  if (!next || next === previous) return
  documentsPage.value = 1
  galleryPage.value = 1
  await loadDocuments()
  if (activeTab.value === 'gallery') await loadGallery()
})

async function loadKnowledgeBases() {
  if (!token.value) return
  knowledgeBasesLoading.value = true
  knowledgeBasesError.value = ''
  try {
    const response = await listEmbedKnowledgeBases(channelId.value, token.value)
    knowledgeBases.value = response?.data ?? []
    const stillAvailable = knowledgeBases.value.some(kb => kb.id === selectedKbId.value)
    selectedKbId.value = stillAvailable ? selectedKbId.value : (knowledgeBases.value[0]?.id ?? '')
    if (!knowledgeBases.value.length) knowledgeBasesError.value = t('embedKnowledge.noKnowledgeBases')
  } catch (error: unknown) {
    knowledgeBasesError.value = errorMessage(error, 'embedKnowledge.loadFailed')
  } finally { knowledgeBasesLoading.value = false }
}

async function loadDocuments() {
  if (!token.value || !selectedKbId.value) return
  documentsLoading.value = true
  documentsError.value = ''
  try {
    const response = await listEmbedKnowledgeDocuments(channelId.value, token.value, selectedKbId.value,
      { page: documentsPage.value, page_size: pageSize, keyword: keyword.value })
    documents.value = response?.data ?? []
    documentsTotal.value = Number(response?.total || 0)
  } catch (error: unknown) {
    documentsError.value = errorMessage(error, 'embedKnowledge.documentsLoadFailed')
  } finally { documentsLoading.value = false }
}

async function loadGallery() {
  if (!token.value || !selectedKbId.value) return
  galleryLoading.value = true
  galleryError.value = ''
  clearGalleryUrls()
  try {
    const response = await listEmbedGalleryImages(channelId.value, token.value, selectedKbId.value,
      { page: galleryPage.value, page_size: galleryPageSize, keyword: keyword.value })
    galleryImages.value = response?.data ?? []
    galleryTotal.value = Number(response?.total || 0)
    await Promise.all(galleryImages.value.map(resolveGalleryImage))
  } catch (error: unknown) {
    galleryError.value = errorMessage(error, 'embedKnowledge.galleryLoadFailed')
  } finally { galleryLoading.value = false }
}

async function resolveGalleryImage(image: EmbedGalleryImage) {
  if (!token.value || !image.url) return
  if (/^https?:\/\//i.test(image.url)) {
    galleryUrls.value = { ...galleryUrls.value, [image.id]: image.url }
    return
  }
  try {
    const blob = await getEmbedImageBlob(channelId.value, token.value, image.url)
    const url = URL.createObjectURL(blob)
    objectUrls.add(url)
    galleryUrls.value = { ...galleryUrls.value, [image.id]: url }
  } catch { /* keep the rest of the gallery usable */ }
}

function clearGalleryUrls() {
  objectUrls.forEach(url => URL.revokeObjectURL(url))
  objectUrls.clear()
  galleryUrls.value = {}
}

function applySearch() {
  keyword.value = keywordInput.value.trim()
  documentsPage.value = 1
  galleryPage.value = 1
  void refreshActive()
}
function refreshActive() { return activeTab.value === 'documents' ? loadDocuments() : loadGallery() }
function onTabChange(value: string | number) {
  activeTab.value = value === 'gallery' ? 'gallery' : 'documents'
  void refreshActive()
}

function appendFiles(files: File[]) {
  const existing = new Set(uploads.value.map(item => `${item.file.name}:${item.file.size}:${item.file.lastModified}`))
  for (const file of files) {
    const identity = `${file.name}:${file.size}:${file.lastModified}`
    if (existing.has(identity)) continue
    existing.add(identity)
    uploads.value.push({ key: `${identity}:${crypto.randomUUID?.() ?? Math.random()}`, file, status: 'pending', progress: 0, error: '' })
  }
}
function handleFileInput(event: Event) {
  const input = event.target as HTMLInputElement
  appendFiles(Array.from(input.files ?? []))
  input.value = ''
}
function handleDrop(event: DragEvent) {
  dragging.value = false
  appendFiles(Array.from(event.dataTransfer?.files ?? []))
}
function removeUpload(key: string) { uploads.value = uploads.value.filter(item => item.key !== key) }

async function uploadAll() {
  if (!selectedKbId.value || !token.value || uploading.value) return
  uploading.value = true
  let successCount = 0
  try {
    for (const entry of uploads.value) {
      if (entry.status !== 'pending') continue
      entry.status = 'uploading'; entry.progress = 0; entry.error = ''
      try {
        await uploadEmbedKnowledgeFile(channelId.value, token.value, selectedKbId.value, entry.file, event => {
          if (event.total) entry.progress = Math.min(99, Math.round(((event.loaded ?? 0) / event.total) * 100))
        })
        entry.progress = 100; entry.status = 'success'; successCount++
      } catch (error: unknown) {
        entry.status = 'error'; entry.error = String((error as { message?: string })?.message || '')
      }
    }
    if (successCount) {
      MessagePlugin.success(t('embedKnowledge.uploadedCount', { count: successCount }))
      documentsPage.value = 1
      await loadDocuments()
    }
  } finally { uploading.value = false }
}

function openEdit(doc: EmbedKnowledgeDocument) {
  editingDocument.value = doc
  editForm.title = documentName(doc)
  editForm.description = doc.description || ''
  editVisible.value = true
}
async function saveEdit() {
  const doc = editingDocument.value
  if (!doc || !token.value || !selectedKbId.value || !editForm.title.trim()) return
  editSaving.value = true
  try {
    await updateEmbedKnowledgeDocument(channelId.value, token.value, selectedKbId.value, doc.id,
      { title: editForm.title.trim(), description: editForm.description.trim() })
    editVisible.value = false
    MessagePlugin.success(t('embedKnowledge.saved'))
    await loadDocuments()
  } catch (error: unknown) {
    MessagePlugin.error(errorMessage(error, 'embedKnowledge.saveFailed'))
  } finally { editSaving.value = false }
}

async function deleteDocument(doc: EmbedKnowledgeDocument) {
  if (!token.value || !selectedKbId.value) return
  try {
    await deleteEmbedKnowledgeDocument(channelId.value, token.value, selectedKbId.value, doc.id)
    MessagePlugin.success(t('embedKnowledge.deleteSubmitted'))
    documents.value = documents.value.filter(item => item.id !== doc.id)
    documentsTotal.value = Math.max(0, documentsTotal.value - 1)
    window.setTimeout(() => void loadDocuments(), 1200)
  } catch (error: unknown) { MessagePlugin.error(errorMessage(error, 'embedKnowledge.deleteFailed')) }
}

async function previewDocument(doc: EmbedKnowledgeDocument) {
  const popup = window.open('about:blank', '_blank')
  try {
    const blob = await fetchDocumentBlob(doc, 'preview')
    const url = URL.createObjectURL(blob)
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
    if (popup) popup.location.href = url
    else window.open(url, '_blank', 'noopener,noreferrer')
  } catch (error: unknown) {
    popup?.close()
    MessagePlugin.error(errorMessage(error, 'embedKnowledge.previewFailed'))
  }
}
async function downloadDocument(doc: EmbedKnowledgeDocument) {
  try {
    const blob = await fetchDocumentBlob(doc, 'download')
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url; anchor.download = documentName(doc); anchor.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (error: unknown) { MessagePlugin.error(errorMessage(error, 'embedKnowledge.downloadFailed')) }
}
function fetchDocumentBlob(doc: EmbedKnowledgeDocument, mode: 'preview' | 'download') {
  if (!token.value || !selectedKbId.value) return Promise.reject(new Error('missing embed session'))
  return getEmbedKnowledgeBlob(channelId.value, token.value, selectedKbId.value, doc.id, mode)
}
function openImage(image: EmbedGalleryImage) { activeImage.value = image; imageViewerVisible.value = true }
function documentName(doc: EmbedKnowledgeDocument) { return doc.title || doc.file_name || doc.id }
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(locale.value)
}
function statusTheme(status?: string): 'success' | 'warning' | 'danger' | 'default' {
  if (status === 'completed') return 'success'
  if (status === 'failed' || status === 'cancelled') return 'danger'
  if (status === 'pending' || status === 'processing' || status === 'finalizing') return 'warning'
  return 'default'
}
function statusLabel(status?: string) { return status ? t(`embedKnowledge.statuses.${status}`) : t('embedKnowledge.statuses.unknown') }
function errorMessage(error: unknown, fallbackKey: string) {
  return String((error as { message?: string })?.message || t(fallbackKey))
}
onBeforeUnmount(clearGalleryUrls)
</script>

<style scoped lang="less">
.embed-knowledge {
  min-height: 100vh; padding: 24px; background: var(--td-bg-color-page); color: var(--td-text-color-primary);
  &__header, &__workspace, &__state { width: min(1180px, 100%); margin: 0 auto; }
  &__header { display: flex; justify-content: space-between; gap: 20px; margin-bottom: 18px; }
  &__header h1 { margin: 0 0 6px; font-size: var(--app-text-3xl); }
  &__header p { margin: 0; color: var(--td-text-color-secondary); }
  &__workspace { display: grid; gap: 16px; padding: 20px; border: 1px solid var(--td-component-stroke); border-radius: var(--app-radius-xl); background: var(--td-bg-color-container); box-shadow: var(--td-shadow-1); }
  &__state, &__loading, &__empty { min-height: 240px; display: flex; align-items: center; justify-content: center; gap: 12px; }
  &__toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  &__kb-select { width: min(300px, 100%); }
  &__search { min-width: 220px; flex: 1; }
  &__upload-panel { display: grid; gap: 14px; padding: 16px; border: 1px solid var(--td-component-stroke); border-radius: var(--app-radius-lg); background: var(--td-bg-color-secondarycontainer); }
  &__dropzone { min-height: 130px; border: 1px dashed var(--td-component-stroke); border-radius: var(--app-radius-lg); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px; cursor: pointer; color: var(--td-text-color-secondary); }
  &__dropzone input { display: none; } &__dropzone strong { color: var(--td-text-color-primary); }
  &__dropzone:hover, &__dropzone.is-dragging { border-color: var(--embed-primary, var(--td-brand-color)); background: var(--td-brand-color-light); }
  &__upload-list { display: grid; gap: 8px; margin: 0; padding: 0; list-style: none; }
  &__upload-list li { display: flex; align-items: flex-start; gap: 10px; padding: 10px; border-radius: var(--app-radius-md); background: var(--td-bg-color-container); }
  &__file-main { min-width: 0; flex: 1; display: grid; gap: 3px; }
  &__file-main > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__file-main small { color: var(--td-text-color-secondary); } &__file-main .is-success { color: var(--td-success-color); } &__file-main .is-error { color: var(--td-error-color); }
  &__upload-actions { display: flex; align-items: center; justify-content: space-between; gap: 16px; color: var(--td-text-color-secondary); font-size: var(--app-text-sm); }
  &__tabs { margin-bottom: -16px; } &__content { display: grid; gap: 16px; min-height: 300px; }
  &__empty { flex-direction: column; color: var(--td-text-color-secondary); }
  &__table-wrap { overflow-x: auto; } &__table { width: 100%; border-collapse: collapse; }
  &__table th, &__table td { padding: 12px; border-bottom: 1px solid var(--td-component-stroke); text-align: left; vertical-align: middle; }
  &__table th { color: var(--td-text-color-secondary); font-size: var(--app-text-sm); font-weight: 500; white-space: nowrap; }
  &__doc-name { min-width: 220px; display: flex; align-items: flex-start; gap: 9px; }
  &__doc-name > div { min-width: 0; display: grid; gap: 4px; }
  &__doc-name strong, &__doc-name small { max-width: 360px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  &__doc-name small { color: var(--td-text-color-secondary); }
  &__row-actions { display: flex; flex-wrap: wrap; gap: 2px; min-width: 220px; }
  &__gallery { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 14px; }
  &__gallery button { overflow: hidden; padding: 0; border: 1px solid var(--td-component-stroke); border-radius: var(--app-radius-lg); background: var(--td-bg-color-secondarycontainer); color: inherit; cursor: pointer; text-align: left; }
  &__gallery img, &__image-loading { width: 100%; height: 160px; object-fit: cover; display: flex; align-items: center; justify-content: center; background: var(--td-bg-color-page); }
  &__gallery span { display: block; overflow: hidden; padding: 10px; text-overflow: ellipsis; white-space: nowrap; }
  &__viewer-image { display: block; max-width: 100%; max-height: 68vh; margin: 0 auto; object-fit: contain; }
}
@media (max-width: 720px) {
  .embed-knowledge { padding: 12px; }
  .embed-knowledge__header { align-items: stretch; flex-direction: column; }
  .embed-knowledge__workspace { padding: 12px; }
  .embed-knowledge__toolbar > * { width: 100%; }
  .embed-knowledge__upload-actions { align-items: stretch; flex-direction: column; }
}
</style>
