import * as FileSystem from "expo-file-system/legacy";
import { Platform } from "react-native";
import { API_URL } from "./config";

export interface UploadProgress {
  uploadedBytes: number;
  totalBytes: number;
  percent: number;
}

function normalizeFileUri(uri: string): string {
  if (Platform.OS === "ios" && !uri.startsWith("file://")) {
    return `file://${uri}`;
  }
  return uri;
}

function parseUploadError(body: string, status: number): string {
  try {
    const data = JSON.parse(body) as { message?: string; error?: string };
    return data.message ?? data.error ?? `Upload failed (${status})`;
  } catch {
    const trimmed = body.trim();
    return trimmed || `Upload failed (${status})`;
  }
}

/** Direct multipart upload — works on React Native (no fetch FormData). */
export async function uploadPageImage(
  uri: string,
  jobId: string,
  pageNumber: number,
  token: string,
  onProgress?: (progress: UploadProgress) => void
): Promise<void> {
  onProgress?.({ uploadedBytes: 0, totalBytes: 100, percent: 0 });

  const result = await FileSystem.uploadAsync(
    `${API_URL}/scan/jobs/${jobId}/pages/${pageNumber}/image`,
    normalizeFileUri(uri),
    {
      uploadType: FileSystem.FileSystemUploadType.MULTIPART,
      fieldName: "file",
      mimeType: "image/jpeg",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

  if (result.status < 200 || result.status >= 300) {
    throw new Error(parseUploadError(result.body, result.status));
  }

  onProgress?.({ uploadedBytes: 100, totalBytes: 100, percent: 100 });
}

export async function uploadInChunks(
  uri: string,
  jobId: string,
  pageNumber: number,
  token: string,
  onProgress?: (progress: UploadProgress) => void
): Promise<void> {
  return uploadPageImage(uri, jobId, pageNumber, token, onProgress);
}

export async function uploadMultiplePages(
  uris: string[],
  jobId: string,
  token: string,
  onProgress?: (percent: number, pageIndex: number, totalPages: number) => void
): Promise<void> {
  const totalPages = uris.length;
  for (let i = 0; i < totalPages; i++) {
    await uploadPageImage(uris[i], jobId, i + 1, token, (pageProgress) => {
      const overall = Math.round(((i + pageProgress.percent / 100) / totalPages) * 100);
      onProgress?.(overall, i, totalPages);
    });
  }
}
