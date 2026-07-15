import { App } from "@capacitor/app";
import { Browser } from "@capacitor/browser";
import { SecureStoragePlugin } from "capacitor-secure-storage-plugin";
import { api } from "@/lib/api";
import {
  clientNativeRedirectUri,
  getNativeAccessToken,
  isNativeClient,
  registerNativeSessionClearer,
  setNativeAccessToken,
} from "@/lib/client-runtime";

const sessionStorageKey = "hl6_native_session";
let initialized = false;

function parseUrl(value: string): URL | null {
  try {
    return new URL(value);
  } catch {
    return null;
  }
}

function isNativeCallback(url: string): boolean {
  const expected = parseUrl(clientNativeRedirectUri);
  const received = parseUrl(url);
  if (!expected || !received) return false;
  return expected.protocol === received.protocol
    && expected.host === received.host
    && expected.pathname === received.pathname;
}

async function persistSession(token: string) {
  setNativeAccessToken(token);
  await SecureStoragePlugin.set({ key: sessionStorageKey, value: token });
}

async function clearPersistedSession() {
  setNativeAccessToken(null);
  try {
    await SecureStoragePlugin.remove({ key: sessionStorageKey });
  } catch {
    // A missing value is already equivalent to a cleared session.
  }
}

async function completeNativeLogin(callbackUrl: string) {
  if (!isNativeCallback(callbackUrl)) return;

  const code = new URL(callbackUrl).searchParams.get("code")?.trim();
  if (!code) return;

  try {
    const response = await api.exchangeNativeAuthCode({ code });
    const token = response.data.access_token?.trim();
    if (!token) throw new Error("Native authentication did not return an access token.");
    await persistSession(token);
    await Browser.close();
    window.location.assign("/dashboard");
  } catch {
    await clearPersistedSession();
    window.location.assign("/?native_auth_error=1");
  }
}

export async function initializeNativeClient() {
  if (!isNativeClient || initialized) return;
  initialized = true;

  registerNativeSessionClearer(() => {
    void clearPersistedSession();
  });

  try {
    const stored = await SecureStoragePlugin.get({ key: sessionStorageKey });
    setNativeAccessToken(stored.value);
  } catch {
    setNativeAccessToken(null);
  }

  await App.addListener("appUrlOpen", ({ url }) => {
    void completeNativeLogin(url);
  });
  const launchUrl = await App.getLaunchUrl();
  if (launchUrl?.url) void completeNativeLogin(launchUrl.url);
}

export async function startNativeSignIn(ref?: string) {
  if (!isNativeClient) return;
  const response = await api.startNativeLogin({
    redirect_uri: clientNativeRedirectUri,
    referral_code: ref,
  });
  await Browser.open({ url: response.data.login_url });
}

export async function signOutNativeClient() {
  let logoutUrl = "";
  try {
    const response = await api.logout();
    logoutUrl = response.data.logout_url?.trim() ?? "";
  } finally {
    await clearPersistedSession();
  }

  if (logoutUrl.startsWith("https://")) {
    await Browser.open({ url: logoutUrl });
  }
  window.location.assign("/");
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
