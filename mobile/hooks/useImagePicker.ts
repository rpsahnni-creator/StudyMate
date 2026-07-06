import { Alert } from "react-native";
import * as ImagePicker from "expo-image-picker";
import * as ImageManipulator from "expo-image-manipulator";
import * as FileSystem from "expo-file-system/legacy";

export interface ImageResult {
  uri: string;
  width: number;
  height: number;
  fileSize: number;
  mimeType: string;
}

const TARGET_MAX_DIMENSION = 1500;
const JPEG_QUALITY = 0.65;
const WARN_SIZE_BYTES = 3 * 1024 * 1024;

const PICKER_OPTIONS: ImagePicker.ImagePickerOptions = {
  mediaTypes: ["images"],
  quality: 1,
  exif: false,
  allowsEditing: true,
};

async function ensureCameraPermission(): Promise<boolean> {
  const current = await ImagePicker.getCameraPermissionsAsync();
  if (current.granted) return true;
  const requested = await ImagePicker.requestCameraPermissionsAsync();
  return requested.granted;
}

async function ensureGalleryPermission(): Promise<boolean> {
  const current = await ImagePicker.getMediaLibraryPermissionsAsync();
  if (current.granted) return true;
  const requested = await ImagePicker.requestMediaLibraryPermissionsAsync();
  return requested.granted;
}

async function compressImage(uri: string, width: number, height: number): Promise<ImageResult> {
  const longest = Math.max(width, height);
  const actions: ImageManipulator.Action[] = [];
  if (longest > TARGET_MAX_DIMENSION) {
    if (width >= height) {
      actions.push({ resize: { width: TARGET_MAX_DIMENSION } });
    } else {
      actions.push({ resize: { height: TARGET_MAX_DIMENSION } });
    }
  }

  const manipulated = await ImageManipulator.manipulateAsync(uri, actions, {
    compress: JPEG_QUALITY,
    format: ImageManipulator.SaveFormat.JPEG,
  });

  const stableUri = `${FileSystem.cacheDirectory}scan-${Date.now()}.jpg`;
  await FileSystem.copyAsync({ from: manipulated.uri, to: stableUri });

  const info = await FileSystem.getInfoAsync(stableUri);
  const fileSize = info.exists && "size" in info && typeof info.size === "number" ? info.size : 0;

  if (fileSize > WARN_SIZE_BYTES) {
    Alert.alert(
      "Large image",
      "Compressed image is still over 3MB. Upload may take longer on slow networks."
    );
  }

  return {
    uri: stableUri,
    width: manipulated.width ?? width,
    height: manipulated.height ?? height,
    fileSize,
    mimeType: "image/jpeg",
  };
}

async function finalizeAsset(asset: ImagePicker.ImagePickerAsset): Promise<ImageResult | null> {
  if (!asset.uri) {
    Alert.alert("Invalid image", "Could not read the selected image.");
    return null;
  }

  const width = asset.width ?? 0;
  const height = asset.height ?? 0;
  if (width <= 0 || height <= 0) {
    const info = await FileSystem.getInfoAsync(asset.uri);
    if (!info.exists) {
      Alert.alert("Invalid image", "Could not read the selected image.");
      return null;
    }
    return compressImage(asset.uri, TARGET_MAX_DIMENSION, TARGET_MAX_DIMENSION);
  }

  return compressImage(asset.uri, width, height);
}

export function useImagePicker() {
  async function pickFromCamera(): Promise<ImageResult | null> {
    const granted = await ensureCameraPermission();
    if (!granted) {
      Alert.alert("Permission required", "Camera access is needed to capture scan pages.");
      return null;
    }

    const result = await ImagePicker.launchCameraAsync(PICKER_OPTIONS);
    if (result.canceled || !result.assets?.length) {
      return null;
    }
    return finalizeAsset(result.assets[0]);
  }

  async function pickFromGallery(): Promise<ImageResult | null> {
    const granted = await ensureGalleryPermission();
    if (!granted) {
      Alert.alert("Permission required", "Gallery access is needed to choose scan pages.");
      return null;
    }

    const result = await ImagePicker.launchImageLibraryAsync(PICKER_OPTIONS);
    if (result.canceled || !result.assets?.length) {
      return null;
    }
    return finalizeAsset(result.assets[0]);
  }

  return { pickFromCamera, pickFromGallery };
}
