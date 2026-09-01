/** Upload adapter injected into the shared FileUpload component. */
export interface FileStoragePort {
  upload(input: UploadInput, signal?: AbortSignal): Promise<UploadResult>
  remove?(file: StoredFile): Promise<void>
  getPreviewUrl?(file: StoredFile): Promise<string>
}

/** File selected by the user before upload. */
export interface UploadInput {
  file: File
  onProgress?: (progress: number) => void
}

/** Stored file metadata returned by a storage adapter. */
export interface StoredFile {
  id: string
  name: string
  size: number
  contentType: string
  url: string
}

/** Result envelope returned after a successful upload. */
export interface UploadResult {
  file: StoredFile
}
