// Points at the Go backend. Locally this falls back to localhost:8080; for
// device builds, EXPO_PUBLIC_API_URL is injected via the "env" block in
// eas.json for the "preview" and "production" build profiles (see eas.json)
// so real builds never ship pointing at localhost.
export const API_URL = process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:8080";
