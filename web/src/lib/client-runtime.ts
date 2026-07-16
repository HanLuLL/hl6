import { Capacitor } from "@capacitor/core";

const bundledCommunicationKey = import.meta.env.VITE_CLIENT_COMMUNICATION_KEY?.trim() ?? "";

export const isNativeClient = import.meta.env.VITE_CAPACITOR_APP === "true" && Capacitor.isNativePlatform();
export const clientVersion = import.meta.env.VITE_CLIENT_VERSION?.trim() ?? "";

let clientCommunicationKey = bundledCommunicationKey;

let nativeAccessToken: string | null = null;
let nativeSessionClearer: (() => void) | undefined;

export function getNativeAccessToken(): string | null {
  return nativeAccessToken;
}

export function getClientCommunicationKey(): string {
  return clientCommunicationKey;
}

export function getBundledClientCommunicationKey(): string {
  return bundledCommunicationKey;
}

export function setClientCommunicationKey(key: string | null) {
  clientCommunicationKey = key?.trim() || "";
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
