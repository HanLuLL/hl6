package cloud.houlang.hl6.client

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

class SecureClientStore(context: Context) {
    private val preferences = EncryptedSharedPreferences.create(
        context,
        "hl6_native_secure_store",
        MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    init {
        val bundledKey = ClientBuild.bundledCommunicationKey
        val persistedKey = preferences.getString(COMMUNICATION_KEY, null)
        if (persistedKey != bundledKey) {
            // A rebuilt APK can carry a rotated key. Its old native session is
            // deliberately bound to the old key hash and must not be reused.
            preferences.edit()
                .putString(COMMUNICATION_KEY, bundledKey)
                .remove(ACCESS_TOKEN)
                .apply()
        }
    }

    fun communicationKey(): String = preferences.getString(COMMUNICATION_KEY, "").orEmpty()

    fun accessToken(): String? = preferences.getString(ACCESS_TOKEN, null)

    fun saveAccessToken(token: String) {
        preferences.edit().putString(ACCESS_TOKEN, token).apply()
    }

    fun clearAccessToken() {
        preferences.edit().remove(ACCESS_TOKEN).apply()
    }

    fun clearAll() {
        preferences.edit().clear().apply()
    }

    private companion object {
        const val COMMUNICATION_KEY = "communication_key"
        const val ACCESS_TOKEN = "access_token"
    }
}
