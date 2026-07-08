import AsyncStorage from '@react-native-async-storage/async-storage';

export type UploadStatus = 'queued' | 'uploading' | 'processing' | 'review' | 'ready' | 'failed';

export interface QueuedUpload {
  id: string;
  fileName: string;
  status: UploadStatus;
  progress: number;
  attempts: number;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
  jobId?: number;
  quizId?: number;
  stageLabel?: string;
  imageUri?: string;
  imageUris?: string[];
  uploadedPageCount?: number;
  payload: {
    mode: string;
    board: string;
    chapter_title?: string;
    accepted_terms: boolean;
    page_no?: number;
  };
}

const STORAGE_KEY = 'studyapp.upload.queue';
const MAX_ATTEMPTS = 5;
const BASE_DELAY_MS = 2000;

export function sortQueueNewestFirst(items: QueuedUpload[]): QueuedUpload[] {
  return [...items].sort((a, b) => {
    const aTime = Date.parse(a.updatedAt || a.createdAt);
    const bTime = Date.parse(b.updatedAt || b.createdAt);
    return bTime - aTime;
  });
}

export async function getQueuedUploads(): Promise<QueuedUpload[]> {
  const raw = await AsyncStorage.getItem(STORAGE_KEY);
  if (!raw) return [];
  try {
    return sortQueueNewestFirst(JSON.parse(raw) as QueuedUpload[]);
  } catch {
    return [];
  }
}

export async function saveQueuedUploads(items: QueuedUpload[]): Promise<void> {
  await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(sortQueueNewestFirst(items)));
}

export async function enqueueUpload(upload: Omit<QueuedUpload, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'attempts'>): Promise<QueuedUpload> {
  const item: QueuedUpload = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    fileName: upload.fileName,
    status: 'queued',
    progress: 0,
    attempts: 0,
    jobId: upload.jobId,
    imageUri: upload.imageUri,
    imageUris: upload.imageUris,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    payload: upload.payload,
  };
  const existing = await getQueuedUploads();
  existing.unshift(item);
  await saveQueuedUploads(existing);
  return item;
}

export async function updateUpload(id: string, patch: Partial<QueuedUpload>): Promise<void> {
  const items = await getQueuedUploads();
  const idx = items.findIndex((item) => item.id === id);
  if (idx === -1) return;
  items[idx] = { ...items[idx], ...patch, updatedAt: new Date().toISOString() };
  await saveQueuedUploads(items);
}

export async function removeUpload(id: string): Promise<void> {
  const items = await getQueuedUploads();
  const next = items.filter((item) => item.id !== id);
  await saveQueuedUploads(next);
}

export function getRetryDelayMs(attempts: number): number {
  return BASE_DELAY_MS * 2 ** Math.min(attempts, 4);
}

export function shouldRetry(attempts: number): boolean {
  return attempts < MAX_ATTEMPTS;
}
