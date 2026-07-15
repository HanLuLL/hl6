import { Capacitor } from "@capacitor/core";

const communicationKey = import.meta.env.VITE_CLIENT_COMMUNICATION_KEY?.trim() ?? "";
const nativeRedirectUri = import.meta.env.VITE_NATIVE_REDIRECT_URI?.trim() ?? "";

export const isNativeClient = import.meta.env.VITE_CAPACITOR_APP === "true" && Capacitor.isNativePlatform();
export const clientVersion = import.meta.env.VITE_CLIENT_VERSION?.trim() ?? "";
export const clientCommunicationKey = communicationKey;
export const clientNativeRedirectUri = nativeRedirectUri;

let nativeAccessToken: string | null = null;
let nativeSessionClearer: (() => void) | undefined;

export function getNativeAccessToken(): string | null {
  return nativeAccessToken;
}

export function setNativeAccessToken(token: string | null) {
  nativeAccessToken = token?.trim() || null;
}

export function registerNativeSessionClearer(clearer: () => void) {
  nativeSessionClearer = clearer;
}

export function clearNativeAccessToken() {
  nativeAccessToken = null;
  nativeSessionClearer?.();
}
