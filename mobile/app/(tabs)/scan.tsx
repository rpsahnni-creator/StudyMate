import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Image,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { router } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import NetInfo from "@react-native-community/netinfo";
import { apiCall, formatApiErrorBody, getAccessToken } from "../../lib/auth";
import { Card, BoardSelect, Checkbox, Field, PrimaryButton, SecondaryButton } from "../../components/ui";
import { BOARD_OPTIONS } from "../../lib/boards";
import { colors, radius, shadow, spacing } from "../../lib/theme";
import { useImagePicker, type ImageResult } from "../../hooks/useImagePicker";
import { uploadMultiplePages, MAX_PAGES_PER_SCAN } from "../../lib/resumableUpload";
import { assertServerReachable } from "../../lib/network";
import { API_URL } from "../../lib/config";
import {
  enqueueUpload,
  getQueuedUploads,
  getRetryDelayMs,
  removeUpload,
  shouldRetry,
  updateUpload,
  sortQueueNewestFirst,
  type QueuedUpload,
} from "../../lib/uploadQueue";
import { pollScanJobStatus, jobStageLabel, isJobReady, setScanJobStrategy, type ScanUploadStatus } from "../../lib/scan";
import { getMySubscription, planDisplayName, scansLabel, type Entitlements } from "../../lib/billing";
import { SkyBackground } from "../../components/SkyBackground";
import { skyScreen } from "../../lib/skyScreen";

const MAX_DRAFT_PAGES = MAX_PAGES_PER_SCAN;

function updateQueueEntry(
  items: QueuedUpload[],
  id: string,
  patch: Partial<QueuedUpload>
): QueuedUpload[] {
  return sortQueueNewestFirst(
    items.map((entry) =>
      entry.id === id ? { ...entry, ...patch, updatedAt: new Date().toISOString() } : entry
    )
  );
}

function isPermanentFailure(message?: string): boolean {
  if (!message) return false;
  const normalized = message.toLowerCase();
  return (
    normalized.includes("scan limit") ||
    normalized.includes("consent") ||
    normalized.includes("sign in")
  );
}

function CaptureButton({
  icon,
  label,
  hint,
  onPress,
  disabled,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  hint: string;
  onPress: () => void;
  disabled?: boolean;
}) {
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      style={({ pressed }) => [
        styles.captureBtn,
        pressed && !disabled ? styles.captureBtnPressed : null,
        disabled ? styles.captureBtnDisabled : null,
      ]}
    >
      <View style={styles.captureIconWrap}>
        <Ionicons name={icon} size={26} color={colors.brandDark} />
      </View>
      <Text style={styles.captureLabel}>{label}</Text>
      <Text style={styles.captureHint}>{hint}</Text>
    </Pressable>
  );
}

export default function ScanScreen() {
  const { pickFromCamera, pickFromGallery } = useImagePicker();
  const [mode, setMode] = useState("chapter");
  const [board, setBoard] = useState("cbse");
  const [chapterTitle, setChapterTitle] = useState("");
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [queue, setQueue] = useState<QueuedUpload[]>([]);
  const [status, setStatus] = useState("Ready for upload");
  const [draftPages, setDraftPages] = useState<ImageResult[]>([]);
  const [compressionStatus, setCompressionStatus] = useState("");
  const [uploadProgress, setUploadProgress] = useState(0);
  const [isUploading, setIsUploading] = useState(false);
  const [isPickingImage, setIsPickingImage] = useState(false);
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);

  const activePolls = useRef<Map<string, { cancel: () => void }>>(new Map());
  const retryUploadRef = useRef<((id: string, silent?: boolean) => Promise<void>) | null>(null);
  const wasOffline = useRef(false);

  useEffect(() => {
    void loadQueue().then((items) => {
      for (const item of items) {
        if (item.status === "processing" && item.jobId) {
          startPolling(item.id, item.jobId);
        }
      }
    });
    void getMySubscription().then(setEntitlements);

    return () => {
      for (const poll of activePolls.current.values()) {
        poll.cancel();
      }
      activePolls.current.clear();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const unsubscribe = NetInfo.addEventListener((state) => {
      const online = Boolean(state.isConnected) && state.isInternetReachable !== false;
      if (!online) {
        wasOffline.current = true;
        return;
      }
      if (!wasOffline.current) return;
      wasOffline.current = false;

      void (async () => {
        const items = await getQueuedUploads();
        for (const item of items) {
          if (item.status !== "failed" || !shouldRetry(item.attempts)) continue;
          if (isPermanentFailure(item.lastError)) continue;
          await new Promise((resolve) => setTimeout(resolve, getRetryDelayMs(item.attempts)));
          await retryUploadRef.current?.(item.id, true);
        }
      })();
    });
    return () => unsubscribe();
  }, []);

  async function loadQueue(): Promise<QueuedUpload[]> {
    const items = await getQueuedUploads();
    setQueue(items);
    return items;
  }

  const startPolling = useCallback((queueId: string, jobId: number) => {
    if (activePolls.current.has(queueId)) return;

    let strategyPromptOpen = false;

    const handleUpdate = (jobStatus: ScanUploadStatus) => {
      const label = jobStageLabel(jobStatus.status);
      setStatus(`Job ${jobId}: ${label}`);
      void updateUpload(queueId, { status: "processing", stageLabel: label });
      setQueue((prev) => updateQueueEntry(prev, queueId, { status: "processing", stageLabel: label }));
    };

    const handleNeedsStrategy = async () => {
      if (strategyPromptOpen) return;
      strategyPromptOpen = true;
      return new Promise<void>((resolve) => {
        Alert.alert(
          "Mixed page detected",
          "Is page me chapter text aur printed questions dono hain. Aap kya chahte ho?",
          [
            {
              text: "Questions nikaalo",
              onPress: () => {
                void setScanJobStrategy(jobId, "extract_questions")
                  .then(() => resolve())
                  .catch((err) => {
                    strategyPromptOpen = false;
                    Alert.alert("Error", err instanceof Error ? err.message : "Strategy save failed");
                    resolve();
                  });
              },
            },
            {
              text: "Chapter se banao",
              onPress: () => {
                void setScanJobStrategy(jobId, "generate_from_chapter")
                  .then(() => resolve())
                  .catch((err) => {
                    strategyPromptOpen = false;
                    Alert.alert("Error", err instanceof Error ? err.message : "Strategy save failed");
                    resolve();
                  });
              },
            },
          ],
          { cancelable: false }
        );
      });
    };

    const poll = pollScanJobStatus(jobId, { onUpdate: handleUpdate, onNeedsStrategy: handleNeedsStrategy });
    activePolls.current.set(queueId, poll);

    poll.promise
      .then(async (finalStatus) => {
        activePolls.current.delete(queueId);
        if (isJobReady(finalStatus.status)) {
          await updateUpload(queueId, { status: "ready", quizId: finalStatus.quiz_id, progress: 100 });
          setQueue((prev) =>
            updateQueueEntry(prev, queueId, {
              status: "ready",
              quizId: finalStatus.quiz_id,
              progress: 100,
            })
          );
          setStatus("Quiz ready!");
          if (finalStatus.quiz_id) {
            Alert.alert("Quiz ready!", "Your scan has been turned into a quiz.", [
              { text: "Later", style: "cancel" },
              {
                text: "Start Quiz",
                onPress: () => router.push(`/quiz/${finalStatus.quiz_id}`),
              },
            ]);
          }
        } else {
          const message = finalStatus.last_error || "Scan processing failed on the server";
          await updateUpload(queueId, { status: "failed", lastError: message, jobId: undefined });
          setQueue((prev) =>
            updateQueueEntry(prev, queueId, {
              status: "failed",
              lastError: message,
              jobId: undefined,
            })
          );
          setStatus(message);
        }
      })
      .catch(async (error) => {
        activePolls.current.delete(queueId);
        const message = error instanceof Error ? error.message : "Could not confirm scan status";
        setStatus(message);
      });
  }, []);

  async function handlePick(fromCamera: boolean) {
    if (!acceptedTerms) {
      Alert.alert("Consent required", "Please accept the educational-use declaration before uploading.");
      return;
    }
    if (draftPages.length >= MAX_DRAFT_PAGES) {
      Alert.alert("Page limit reached", `You can add up to ${MAX_DRAFT_PAGES} pages per scan.`);
      return;
    }

    setIsPickingImage(true);
    setCompressionStatus("Opening crop editor...");
    try {
      const image = fromCamera ? await pickFromCamera() : await pickFromGallery();
      if (!image) {
        setCompressionStatus("");
        return;
      }

      setDraftPages((prev) => [...prev, image]);
      setCompressionStatus(
        `Page ${draftPages.length + 1} ready · ${image.width}×${image.height} · ${Math.round(image.fileSize / 1024)} KB`
      );
      setUploadProgress(0);
      setStatus(`${draftPages.length + 1} page(s) ready — add more or upload all`);
    } finally {
      setIsPickingImage(false);
    }
  }

  function removeDraftPage(index: number) {
    setDraftPages((prev) => prev.filter((_, i) => i !== index));
    setStatus("Ready for upload");
  }

  async function startUpload(existingJobId?: number, existingQueueId?: string) {
    const pagesToUpload = draftPages;
    if (pagesToUpload.length === 0) {
      Alert.alert("No pages", "Take a photo or choose from gallery first.");
      return;
    }
    if (!acceptedTerms) {
      Alert.alert("Consent required", "Please accept the educational-use declaration before uploading.");
      return;
    }

    const net = await NetInfo.fetch();
    if (net.type === "cellular") {
      Alert.alert(
        "Cellular network",
        "You are on mobile data. The upload will continue, but it may use significant data."
      );
    }

    const token = await getAccessToken();
    if (!token) {
      Alert.alert("Sign in required", "Please log in before uploading scan pages.");
      return;
    }

    try {
      await assertServerReachable();
    } catch (error) {
      const message = error instanceof Error ? error.message : "Server unreachable";
      Alert.alert("Server connect nahi ho paaya", message);
      setStatus(message);
      return;
    }

    setIsUploading(true);
    setUploadProgress(0);
    const pageCount = pagesToUpload.length;
    const primaryUri = pagesToUpload[0].uri;
    const allUris = pagesToUpload.map((p) => p.uri);

    let queueItem: QueuedUpload;
    if (existingQueueId) {
      const items = await getQueuedUploads();
      const found = items.find((entry) => entry.id === existingQueueId);
      if (!found) {
        setIsUploading(false);
        Alert.alert("Retry unavailable", "Upload queue entry not found.");
        return;
      }
      queueItem = found;
      await updateUpload(queueItem.id, { status: "uploading", progress: 0, imageUri: primaryUri, imageUris: allUris });
    } else {
      queueItem = await enqueueUpload({
        fileName: `scan-${Date.now()}-${pageCount}pg.jpg`,
        imageUri: primaryUri,
        imageUris: allUris,
        payload: {
          mode,
          board,
          chapter_title: chapterTitle.trim(),
          accepted_terms: acceptedTerms,
          page_no: pageCount,
        },
      });
      setQueue((prev) => sortQueueNewestFirst([queueItem, ...prev]));
    }

    try {
      let jobId = existingJobId ?? queueItem.jobId;
      if (!jobId) {
        setStatus("Creating scan job...");
        const response = await apiCall("/scan/jobs", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            mode,
            board,
            chapter_title: chapterTitle.trim() || undefined,
            accepted_terms: acceptedTerms,
            page_no: pageCount,
          }),
        });
        const data = (await response.json()) as {
          error?: string;
          message?: string;
          details?: Record<string, string>;
          job?: { id?: number };
        };
        if (!response.ok) {
          if (response.status === 429) {
            throw new Error("Daily scan limit reached. Upgrade your plan for more scans.");
          }
          throw new Error(formatApiErrorBody(data));
        }
        jobId = data.job?.id;
        if (!jobId) {
          throw new Error("Scan job was not created");
        }
        await updateUpload(queueItem.id, { jobId, imageUri: primaryUri, imageUris: allUris });
      }

      await updateUpload(queueItem.id, { status: "uploading", progress: 0 });
      setStatus(`Uploading ${pageCount} page(s) to job ${jobId}...`);

      const urisForUpload = queueItem.imageUris?.length ? queueItem.imageUris : allUris;
      const resumeFrom = queueItem.uploadedPageCount ?? 0;

      if (resumeFrom > 0) {
        setStatus(`Resuming upload from page ${resumeFrom + 1} of ${urisForUpload.length}...`);
      }

      const uploadedCount = await uploadMultiplePages(
        urisForUpload,
        String(jobId),
        token,
        (percent, pageIndex, total) => {
          setUploadProgress(percent);
          setStatus(`Uploading page ${pageIndex + 1} of ${total}…`);
          void updateUpload(queueItem.id, { status: "uploading", progress: percent });
          setQueue((prev) =>
            updateQueueEntry(prev, queueItem.id, { status: "uploading", progress: percent, jobId })
          );
        },
        {
          startFromPageIndex: resumeFrom,
          onPageComplete: (pageIndex) => {
            void updateUpload(queueItem.id, { uploadedPageCount: pageIndex + 1, progress: undefined });
          },
        }
      );

      await updateUpload(queueItem.id, {
        status: "processing",
        progress: 100,
        uploadedPageCount: uploadedCount,
      });
      setQueue((prev) =>
        updateQueueEntry(prev, queueItem.id, { status: "processing", progress: 100 })
      );
      setUploadProgress(100);
      setDraftPages([]);
      setStatus(`Upload complete — job ${jobId} queued for OCR`);
      startPolling(queueItem.id, jobId);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Upload failed";
      await updateUpload(queueItem.id, {
        status: "failed",
        attempts: queueItem.attempts + 1,
        lastError: message,
      });
      setQueue((prev) =>
        updateQueueEntry(prev, queueItem.id, {
          status: "failed",
          attempts: queueItem.attempts + 1,
          lastError: message,
        })
      );
      setStatus(message);
      Alert.alert("Upload failed", message);
    } finally {
      setIsUploading(false);
    }
  }

  async function retryUpload(id: string, silent = false) {
    const items = await getQueuedUploads();
    const item = items.find((entry) => entry.id === id);
    if (!item) {
      if (!silent) Alert.alert("Retry unavailable", "Upload queue entry not found.");
      return;
    }
    const uris = item.imageUris?.length ? item.imageUris : item.imageUri ? [item.imageUri] : [];
    if (uris.length === 0) {
      if (!silent) Alert.alert("Retry unavailable", "Select the pages again before retrying.");
      return;
    }
    setDraftPages(
      uris.map((uri) => ({
        uri,
        width: 0,
        height: 0,
        fileSize: 0,
        mimeType: "image/jpeg",
      }))
    );
    if (item.payload.chapter_title) {
      setChapterTitle(item.payload.chapter_title);
    }
    await updateUpload(id, { status: "queued", progress: 0, uploadedPageCount: item.uploadedPageCount ?? 0 });
    setQueue((prev) => updateQueueEntry(prev, id, { status: "queued", progress: 0 }));
    await startUpload(item.jobId, id);
  }
  retryUploadRef.current = retryUpload;

  async function clearUpload(id: string) {
    await removeUpload(id);
    setQueue((prev) => prev.filter((entry) => entry.id !== id));
  }

  const networkHint = useMemo(() => {
    return Platform.OS === "ios" || Platform.OS === "android"
      ? `Up to ${MAX_DRAFT_PAGES} pages · auto-retry · same Wi‑Fi as PC`
      : "Low-bandwidth mode enabled";
  }, []);

  const statusColor = status.toLowerCase().includes("fail") || status.toLowerCase().includes("error")
    ? colors.danger
    : status.toLowerCase().includes("complete") || status.toLowerCase().includes("ready")
      ? colors.success
      : colors.text;

  const canCapture = !isUploading && !isPickingImage && draftPages.length < MAX_DRAFT_PAGES;

  return (
    <SkyBackground>
    <ScrollView style={styles.screen} contentContainerStyle={styles.container}>
      <View style={[skyScreen.heroCard, styles.hero]}>
        <View style={styles.heroContent}>
          <View style={styles.heroIconWrap}>
            <Ionicons name="scan-outline" size={28} color={colors.white} />
          </View>
          <View style={styles.heroText}>
            <Text style={styles.heroTitle}>Scan pages</Text>
            <Text style={styles.heroSub}>{networkHint}</Text>
          </View>
        </View>
        {entitlements ? (
          <View style={styles.planPill}>
            <View style={styles.planBadge}>
              <Text style={styles.planBadgeText}>{planDisplayName(entitlements.plan)}</Text>
            </View>
            <Text style={styles.planScans}>{scansLabel(entitlements)}</Text>
          </View>
        ) : null}
      </View>

      <Card glass style={styles.formCard}>
        <View style={styles.sectionHead}>
          <Text style={styles.sectionTitle}>Chapter details</Text>
          <Text style={styles.sectionSub}>NCERT & state-board chapters</Text>
        </View>

        <View style={styles.sectionHead}>
          <Text style={styles.sectionTitle}>Scan mode</Text>
          <Text style={styles.sectionSub}>Chapter se naye questions ya book ke printed questions</Text>
        </View>

        <View style={styles.modeRow}>
          <Pressable
            onPress={() => setMode("chapter")}
            style={[styles.modeChip, mode === "chapter" ? styles.modeChipActive : null]}
          >
            <Ionicons name="book-outline" size={18} color={mode === "chapter" ? colors.white : colors.brandDark} />
            <Text style={[styles.modeChipText, mode === "chapter" ? styles.modeChipTextActive : null]}>
              Chapter scan
            </Text>
            <Text style={[styles.modeChipHint, mode === "chapter" ? styles.modeChipHintActive : null]}>
              Page se related questions (flexible count)
            </Text>
          </Pressable>
          <Pressable
            onPress={() => setMode("existing_questions")}
            style={[styles.modeChip, mode === "existing_questions" ? styles.modeChipActive : null]}
          >
            <Ionicons name="help-circle-outline" size={18} color={mode === "existing_questions" ? colors.white : colors.brandDark} />
            <Text style={[styles.modeChipText, mode === "existing_questions" ? styles.modeChipTextActive : null]}>
              Question scan
            </Text>
            <Text style={[styles.modeChipHint, mode === "existing_questions" ? styles.modeChipHintActive : null]}>
              Printed questions detect
            </Text>
          </Pressable>
        </View>

        <BoardSelect
          label="Board"
          value={board}
          options={BOARD_OPTIONS}
          onChange={setBoard}
        />

        <Field label="Chapter" value={chapterTitle} onChangeText={setChapterTitle} placeholder="e.g. Interjections" />

        <Text style={styles.stubHint}>
          Production scan: backend .env me OCR_PROVIDER=gemini_vision + GEMINI_API_KEY set karo. Chapter name likhna helpful hai.
        </Text>

        <Checkbox
          checked={acceptedTerms}
          onToggle={() => setAcceptedTerms((prev) => !prev)}
          label="I confirm I am scanning legally acquired NCERT or state-board material for personal educational use."
        />

        <View style={styles.sectionHead}>
          <Text style={styles.sectionTitle}>Capture pages</Text>
          <Text style={styles.sectionSub}>
            Crop each page · up to {MAX_DRAFT_PAGES} pages · upload all at once
          </Text>
        </View>

        <View style={styles.row}>
          <CaptureButton
            icon="camera-outline"
            label="Camera"
            hint="Take & crop"
            onPress={() => void handlePick(true)}
            disabled={!canCapture}
          />
          <CaptureButton
            icon="images-outline"
            label="Gallery"
            hint="Pick & crop"
            onPress={() => void handlePick(false)}
            disabled={!canCapture}
          />
        </View>

        {isPickingImage ? (
          <View style={styles.processingBanner}>
            <ActivityIndicator size="small" color={colors.brand} />
            <Text style={styles.processingText}>{compressionStatus || "Processing image..."}</Text>
          </View>
        ) : null}

        {draftPages.length > 0 ? (
          <View style={styles.pagesWrap}>
            <View style={styles.pagesHeader}>
              <Text style={styles.pagesCount}>
                {draftPages.length} / {MAX_DRAFT_PAGES} pages
              </Text>
              {draftPages.length < MAX_DRAFT_PAGES ? (
                <Text style={styles.addMoreHint}>Tap Camera or Gallery to add more</Text>
              ) : null}
            </View>
            <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.pagesRow}>
              {draftPages.map((page, index) => (
                <View key={`${page.uri}-${index}`} style={styles.pageThumb}>
                  <Image source={{ uri: page.uri }} style={styles.pageThumbImg} resizeMode="cover" />
                  <View style={styles.pageThumbBadge}>
                    <Text style={styles.pageThumbBadgeText}>P{index + 1}</Text>
                  </View>
                  {!isUploading ? (
                    <Pressable style={styles.pageRemoveBtn} onPress={() => removeDraftPage(index)}>
                      <Ionicons name="close" size={14} color={colors.white} />
                    </Pressable>
                  ) : null}
                </View>
              ))}
            </ScrollView>
            {compressionStatus ? <Text style={styles.meta}>{compressionStatus}</Text> : null}
          </View>
        ) : (
          <View style={styles.emptyPreview}>
            <Ionicons name="document-text-outline" size={32} color={colors.textSubtle} />
            <Text style={styles.emptyPreviewText}>No pages yet</Text>
            <Text style={styles.emptyPreviewHint}>Take a photo or pick from gallery — crop each page</Text>
          </View>
        )}

        <PrimaryButton
          title={
            isUploading
              ? "Uploading..."
              : draftPages.length > 0
                ? `Upload all ${draftPages.length} page${draftPages.length > 1 ? "s" : ""}`
                : "Upload pages"
          }
          icon="cloud-upload-outline"
          onPress={() => void startUpload()}
          disabled={isUploading || draftPages.length === 0}
          loading={isUploading}
        />

        <View style={styles.progressSection}>
          <View style={styles.progressTrack}>
            <View style={[styles.progressFill, { width: `${uploadProgress}%` }]} />
          </View>
          <View style={styles.statusRow}>
            <Text style={[styles.status, { color: statusColor }]} numberOfLines={2}>
              {status}
            </Text>
            <Text style={styles.progressPct}>{uploadProgress}%</Text>
          </View>
        </View>
        <Text style={styles.serverMeta} numberOfLines={1}>
          Server: {API_URL}
        </Text>
      </Card>

      {queue.length > 0 ? (
        <View style={styles.queueSection}>
          <Text style={styles.queueHeading}>Recent quizzes</Text>
          {queue.map((item) => (
            <Card glass key={item.id} style={styles.item}>
              <View style={styles.itemHead}>
                <Text style={styles.itemName} numberOfLines={1}>
                  {item.fileName}
                </Text>
                <StatusPill status={item.status} />
              </View>
              <View style={styles.miniTrack}>
                <View style={[styles.miniFill, { width: `${item.progress}%` }]} />
              </View>
              {item.status === "processing" ? (
                <View style={styles.processingRow}>
                  <ActivityIndicator size="small" color={colors.brand} />
                  <Text style={styles.meta}>{item.stageLabel ?? "Processing…"}</Text>
                </View>
              ) : null}
              {item.lastError ? <Text style={styles.errorText}>{item.lastError}</Text> : null}
              <View style={styles.itemActions}>
                {item.status === "ready" && item.quizId ? (
                  <PrimaryButton
                    title="Start Quiz"
                    icon="play-outline"
                    onPress={() => router.push(`/quiz/${item.quizId}`)}
                    style={styles.flexBtn}
                  />
                ) : null}
                {item.status === "failed" ? (
                  <SecondaryButton
                    title="Retry"
                    icon="refresh-outline"
                    onPress={() => void retryUpload(item.id)}
                    style={styles.flexBtn}
                  />
                ) : null}
                <SecondaryButton
                  title="Remove"
                  icon="trash-outline"
                  onPress={() => void clearUpload(item.id)}
                  style={styles.flexBtn}
                />
              </View>
            </Card>
          ))}
        </View>
      ) : null}
    </ScrollView>
    </SkyBackground>
  );
}

function StatusPill({ status }: { status: string }) {
  const map: Record<string, { bg: string; color: string }> = {
    failed: { bg: colors.dangerBg, color: colors.danger },
    processing: { bg: colors.warningBg, color: colors.warning },
    uploading: { bg: colors.brandSoft, color: colors.brandDark },
    queued: { bg: colors.surfaceAlt, color: colors.textMuted },
    ready: { bg: colors.successBg, color: colors.success },
  };
  const theme = map[status] ?? { bg: colors.successBg, color: colors.success };
  return (
    <View style={[styles.pill, { backgroundColor: theme.bg }]}>
      <Text style={[styles.pillText, { color: theme.color }]}>{status}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  container: { paddingHorizontal: spacing.xl, paddingTop: spacing.lg, paddingBottom: 40, gap: spacing.lg },
  hero: {
    marginBottom: 0,
  },
  heroContent: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  heroIconWrap: {
    width: 52,
    height: 52,
    borderRadius: 16,
    backgroundColor: colors.brand,
    alignItems: "center",
    justifyContent: "center",
    ...shadow.brand,
  },
  heroText: { flex: 1, gap: 4 },
  heroTitle: { fontSize: 24, fontWeight: "800", color: "#0F172A", letterSpacing: -0.3 },
  heroSub: { fontSize: 13.5, color: "#64748B", lineHeight: 19 },
  planPill: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    marginTop: spacing.sm,
    backgroundColor: "rgba(240, 180, 41, 0.12)",
    borderRadius: radius.full,
    paddingVertical: 8,
    paddingHorizontal: 14,
    alignSelf: "flex-start",
    borderWidth: 1,
    borderColor: "rgba(240, 180, 41, 0.28)",
  },
  planBadge: {
    backgroundColor: colors.white,
    paddingHorizontal: 10,
    paddingVertical: 3,
    borderRadius: radius.full,
    borderWidth: 1,
    borderColor: "#FCD34D",
  },
  planBadgeText: { fontSize: 11, fontWeight: "800", color: "#B45309" },
  planScans: { fontSize: 13, fontWeight: "700", color: "#334155" },
  formCard: { gap: spacing.lg, ...shadow.md },
  sectionHead: { gap: 2 },
  sectionTitle: { fontSize: 16, fontWeight: "800", color: colors.text },
  sectionSub: { fontSize: 13, color: colors.textMuted },
  modeRow: { flexDirection: "row", gap: spacing.md },
  modeChip: {
    flex: 1,
    backgroundColor: colors.surfaceAlt,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.md,
    gap: 4,
    ...shadow.sm,
  },
  modeChipActive: { backgroundColor: colors.brandDark, borderColor: colors.brandDark },
  modeChipText: { fontSize: 14, fontWeight: "800", color: colors.text },
  modeChipTextActive: { color: colors.white },
  modeChipHint: { fontSize: 11, fontWeight: "600", color: colors.textMuted, lineHeight: 15 },
  modeChipHintActive: { color: "rgba(255,255,255,0.85)" },
  stubHint: {
    fontSize: 12,
    lineHeight: 17,
    color: colors.textMuted,
    backgroundColor: colors.surfaceAlt,
    padding: spacing.md,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
  },
  row: { flexDirection: "row", gap: spacing.md },
  half: { flex: 1 },
  flexBtn: { flex: 1 },
  captureBtn: {
    flex: 1,
    backgroundColor: colors.surfaceAlt,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.border,
    paddingVertical: spacing.lg,
    paddingHorizontal: spacing.md,
    alignItems: "center",
    gap: 6,
    ...shadow.sm,
  },
  captureBtnPressed: { opacity: 0.92, transform: [{ scale: 0.98 }] },
  captureBtnDisabled: { opacity: 0.45 },
  captureIconWrap: {
    width: 52,
    height: 52,
    borderRadius: 16,
    backgroundColor: colors.brandSoft,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1,
    borderColor: colors.brandSoftBorder,
  },
  captureLabel: { fontSize: 15, fontWeight: "800", color: colors.text },
  captureHint: { fontSize: 12, color: colors.textMuted, fontWeight: "600" },
  processingBanner: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    backgroundColor: colors.brandSoft,
    borderRadius: radius.md,
    padding: spacing.md,
    borderWidth: 1,
    borderColor: colors.brandSoftBorder,
  },
  processingText: { flex: 1, fontSize: 13.5, fontWeight: "600", color: colors.brandDark },
  previewBox: { gap: 8, position: "relative" },
  preview: {
    width: "100%",
    height: 240,
    borderRadius: radius.lg,
    backgroundColor: colors.surfaceAlt,
    borderWidth: 1,
    borderColor: colors.border,
  },
  previewBadge: {
    position: "absolute",
    top: 12,
    right: 12,
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    backgroundColor: "rgba(15,23,42,0.72)",
    paddingHorizontal: 10,
    paddingVertical: 5,
    borderRadius: radius.full,
  },
  previewBadgeText: { color: colors.white, fontSize: 12, fontWeight: "700" },
  pagesWrap: { gap: 10 },
  pagesHeader: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", gap: 8 },
  pagesCount: { fontSize: 14, fontWeight: "800", color: colors.text },
  addMoreHint: { fontSize: 12, fontWeight: "600", color: colors.brandDark, flex: 1, textAlign: "right" },
  pagesRow: { gap: 12, paddingVertical: 4 },
  pageThumb: {
    width: 120,
    height: 160,
    borderRadius: radius.md,
    overflow: "hidden",
    backgroundColor: colors.surfaceAlt,
    borderWidth: 1,
    borderColor: colors.border,
    position: "relative",
  },
  pageThumbImg: { width: "100%", height: "100%" },
  pageThumbBadge: {
    position: "absolute",
    top: 8,
    left: 8,
    backgroundColor: "rgba(15,23,42,0.75)",
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: radius.full,
  },
  pageThumbBadgeText: { color: colors.white, fontSize: 11, fontWeight: "800" },
  pageRemoveBtn: {
    position: "absolute",
    top: 8,
    right: 8,
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: colors.danger,
    alignItems: "center",
    justifyContent: "center",
  },
  emptyPreview: {
    alignItems: "center",
    justifyContent: "center",
    gap: 6,
    paddingVertical: spacing.xxl,
    backgroundColor: colors.surfaceAlt,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.border,
    borderStyle: "dashed",
  },
  emptyPreviewText: { fontSize: 15, fontWeight: "700", color: colors.textMuted },
  emptyPreviewHint: { fontSize: 12.5, color: colors.textSubtle, textAlign: "center", paddingHorizontal: spacing.xl },
  meta: { color: colors.textMuted, fontSize: 13, lineHeight: 18 },
  serverMeta: { color: colors.textSubtle, fontSize: 11, marginTop: spacing.xs },
  progressSection: { gap: 8 },
  progressTrack: {
    height: 8,
    borderRadius: radius.full,
    backgroundColor: colors.surfaceAlt,
    overflow: "hidden",
  },
  progressFill: { height: "100%", backgroundColor: colors.brand, borderRadius: radius.full },
  statusRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "flex-start", gap: 12 },
  status: { fontWeight: "700", fontSize: 14, flex: 1, lineHeight: 20 },
  progressPct: { fontWeight: "800", fontSize: 14, color: colors.brandDark },
  queueSection: { marginTop: spacing.sm, gap: spacing.md },
  queueHeading: { fontSize: 16, fontWeight: "800", color: colors.text },
  item: { gap: 10 },
  itemHead: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: 10 },
  itemName: { flex: 1, fontWeight: "700", color: colors.text },
  miniTrack: { height: 6, borderRadius: radius.full, backgroundColor: colors.surfaceAlt, overflow: "hidden" },
  miniFill: { height: "100%", backgroundColor: colors.brand, borderRadius: radius.full },
  processingRow: { flexDirection: "row", alignItems: "center", gap: 8 },
  errorText: { color: colors.danger, fontSize: 13 },
  itemActions: { flexDirection: "row", gap: spacing.md },
  pill: { paddingHorizontal: 10, paddingVertical: 3, borderRadius: radius.full },
  pillText: { fontSize: 12, fontWeight: "700", textTransform: "capitalize" },
});
