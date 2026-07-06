import AsyncStorage from "@react-native-async-storage/async-storage";

const PROFILE_IMAGE_KEY = "studyapp_profile_image";

export async function getProfileImageUri(): Promise<string | null> {
  return AsyncStorage.getItem(PROFILE_IMAGE_KEY);
}

export async function saveProfileImageUri(uri: string): Promise<void> {
  await AsyncStorage.setItem(PROFILE_IMAGE_KEY, uri);
}

export async function clearProfileImageUri(): Promise<void> {
  await AsyncStorage.removeItem(PROFILE_IMAGE_KEY);
}

export function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "S";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0]}${parts[parts.length - 1][0]}`.toUpperCase();
}
