import { Browser } from "@capacitor/browser";
import { App } from "@capacitor/app";
import { SecureStoragePlugin } from "capacitor-secure-storage-plugin";
import { api } from "@/lib/api";
import {
  getBundledClientCommunicationKey,
  getNativeAccessToken,
  isNativeClient,
  registerNativeSessionClearer,
  setClientCommunicationKey,
  setNativeAccessToken,
} from "@/lib/client-runtime";

const sessionStorageKey = "hl6_native_session";
const communicationKeyStorageKey = "hl6_native_communication_key";
let initialized = false;

async function persistSession(token: string) {
  setNativeAccessToken(token);
  await SecureStoragePlugin.set({ key: sessionStorageKey, value: token });
}

async function clearPersistedSession() {
  setNativeAccessToken(null);
  try {
    await SecureStoragePlugin.remove({ key: sessionStorageKey });
  } catch {
    // A missing secure-storage item is already a cleared session.
  }
}

export async function initializeNativeClient() {
  if (!isNativeClient || initialized) return;
  initialized = true;
  registerNativeSessionClearer(() => {
    void clearPersistedSession();
  });

  const bundledKey = getBundledClientCommunicationKey();
  try {
    const storedKey = await SecureStoragePlugin.get({ key: communicationKeyStorageKey });
    if (bundledKey && storedKey.value !== bundledKey) {
      setClientCommunicationKey(bundledKey);
      await SecureStoragePlugin.set({ key: communicationKeyStorageKey, value: bundledKey });
    } else {
      setClientCommunicationKey(storedKey.value);
    }
  } catch {
    setClientCommunicationKey(bundledKey);
    if (bundledKey) {
      try {
        await SecureStoragePlugin.set({ key: communicationKeyStorageKey, value: bundledKey });
      } catch {
        // The bundled identifier still allows the client to show a recoverable update prompt.
      }
    }
  }

  try {
    const stored = await SecureStoragePlugin.get({ key: sessionStorageKey });
    setNativeAccessToken(stored.value);
  } catch {
    setNativeAccessToken(null);
  }
}

export async function signInNative(email: string, password: string) {
  if (!isNativeClient) {
    throw new Error("Native sign-in is unavailable outside the packaged client.");
  }
  const response = await api.login({ email, password });
  const token = response.data.access_token?.trim();
  if (!token) {
    throw new Error("Native authentication did not return an access token.");
  }
  await persistSession(token);
  return response.data;
}

export async function signOutNativeClient() {
  try {
    await api.logout();
  } finally {
    await clearPersistedSession();
  }
  window.location.assign("/login");
}

export async function openNativeExternalUrl(url: string) {
  if (!isNativeClient) {
    window.location.assign(url);
    return;
  }
  await Browser.open({ url });
}

export function hasNativeSession() {
  return Boolean(getNativeAccessToken());
}

// Deep link handling for custom scheme URLs (hl6://activate?token=xxx or hl6://reset-password?token=xxx)
export type DeepLinkHandler = (path: string, params: Record<string, string>) => void;

let deepLinkHandler: DeepLinkHandler | null = null;

/**
 * Sets up deep link listener for the native client.
 * Handles URLs like:
 * - hl6://activate?token=xxx
 * - hl6://reset-password?token=xxx
 *
 * The handler will be called with the path (e.g., "activate") and query params.
 */
export function setupDeepLinkListener(handler: DeepLinkHandler) {
  if (!isNativeClient) return;

  deepLinkHandler = handler;

  App.addListener("appUrlOpen", (event) => {
    const url = event.url;
    if (!url || !url.startsWith("hl6://")) return;

    try {
      // Parse the deep link URL
      // Format: hl6://path?param1=value1&param2=value2
      const urlObj = new URL(url.replace("hl6://", "http://placeholder/"));
      const path = urlObj.pathname.replace("/", "");
      const params: Record<string, string> = {};

      urlObj.searchParams.forEach((value, key) => {
        params[key] = value;
      });

      if (deepLinkHandler) {
        deepLinkHandler(path, params);
      }
    } catch {
      // Invalid URL format, ignore
    }
  });
}

/**
 * Removes the deep link listener.
 */
export function removeDeepLinkListener() {
  deepLinkHandler = null;
}