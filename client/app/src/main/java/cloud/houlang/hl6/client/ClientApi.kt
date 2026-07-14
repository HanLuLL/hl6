package cloud.houlang.hl6.client

import android.net.Uri
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObjectBuilder
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.util.UUID
import java.util.concurrent.TimeUnit

@Serializable
data class ApiEnvelope<T>(val code: Int, val message: String = "", val data: T? = null)

@Serializable
data class OffsetApiEnvelope<T>(
    val code: Int,
    val message: String = "",
    val data: T? = null,
    val total: Long = 0,
    val offset: Int = 0,
    val limit: Int = 20,
)

data class OffsetPage<T>(val items: T, val total: Long, val offset: Int, val limit: Int)

@Serializable
data class ClientVersion(
    val latest_version: String = "",
    val force_update: Boolean = false,
    val update_notice: String = "",
    val update_url: String = "",
    val update_available: Boolean = false,
)

@Serializable
data class ClientUser(
    val id: Long = 0,
    val email: String = "",
    val name: String = "",
    val avatar_url: String = "",
    val bio: String = "",
    val website: String = "",
    val role: String = "user",
    val is_banned: Boolean = false,
    val banned_reason: String = "",
    val banned_at: String? = null,
    val banned_until: String? = null,
)

@Serializable
data class MePayload(val user: ClientUser, val credits: Double = 0.0)

@Serializable
data class ClientDomain(
    val id: Long = 0,
    val name: String = "",
    val description: String = "",
    val credit_cost: Double = 0.0,
    val is_active: Boolean = true,
    val migration_read_only: Boolean = false,
)

@Serializable
data class ClientDnsRecord(
    val id: Long = 0,
    val subdomain_id: Long = 0,
    val type: String = "",
    val name: String = "",
    val content: String = "",
    val ttl: Int = 1,
    val proxied: Boolean = false,
    val status: String = "active",
)

@Serializable
data class ClientSubdomain(
    val id: Long = 0,
    val domain_id: Long = 0,
    val fqdn: String = "",
    val name: String = "",
    val claim_cost: Double = 0.0,
    val status: String = "active",
    val suspended_reason: String = "",
    val domain: ClientDomain? = null,
    val dns_records: List<ClientDnsRecord> = emptyList(),
)

@Serializable
data class ClientCreditTransaction(
    val id: Long = 0,
    val amount: Double = 0.0,
    val type: String = "",
    val description_key: String = "",
    val balance_after: Double = 0.0,
    val created_at: String = "",
)

@Serializable
data class ClientCreditBalance(
    val balance: Double = 0.0,
    val transactions: List<ClientCreditTransaction> = emptyList(),
)

@Serializable
data class ClientDailyCheckinStatus(
    val enabled: Boolean = false,
    val reward: Double = 0.0,
    val claimed_today: Boolean = false,
    val checkin_date: String = "",
)

@Serializable
data class ClientNotification(
    val id: Long = 0,
    val title: String = "",
    val content: String = "",
    val type: String = "normal",
    val is_read: Boolean = false,
    val created_at: String = "",
)

@Serializable
data class ClientFriendLink(
    val id: Long = 0,
    val name: String = "",
    val url: String = "",
    val description: String = "",
    val logo_url: String = "",
)

@Serializable
data class ProfilePayload(val user: ClientUser)

@Serializable
data class NativeLoginPayload(val login_url: String)

@Serializable
data class NativeExchangePayload(val access_token: String, val expires_in: Long)

class ClientApiException(val statusCode: Int, override val message: String) : IOException(message)

class ClientApi(private val store: SecureClientStore) {
    private val json = Json {
        ignoreUnknownKeys = true
    }
    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .callTimeout(40, TimeUnit.SECONDS)
        .addInterceptor { chain ->
            val original = chain.request()
            val request = original.newBuilder()
                .header("Accept", "application/json")
                .header("X-HL6-Client-Key", store.communicationKey())
                .apply {
                    store.accessToken()?.takeIf { it.isNotBlank() }?.let { header("Authorization", "Bearer $it") }
                    if (original.method !in SAFE_METHODS) {
                        header("X-Idempotency-Key", UUID.randomUUID().toString())
                    }
                }
                .build()
            chain.proceed(request)
        }
        .build()

    suspend fun getVersion(): ClientVersion =
        request("/client/version?current_version=${Uri.encode(ClientBuild.version)}")

    suspend fun startNativeLogin(): NativeLoginPayload =
        request("/auth/native/start", "POST", jsonBody { put("redirect_uri", ClientBuild.nativeRedirectUri) })

    suspend fun exchangeNativeCode(code: String): NativeExchangePayload =
        request("/auth/native/exchange", "POST", jsonBody { put("code", code) })

    suspend fun getMe(): MePayload = request("/auth/me")

    suspend fun getSubdomains(): List<ClientSubdomain> = request("/subdomains")

    suspend fun getSubdomain(id: Long): ClientSubdomain = request("/subdomains/$id")

    suspend fun claimSubdomain(domainID: Long, name: String) {
        request<JsonElement>("/subdomains", "POST", jsonBody {
            put("domain_id", domainID)
            put("name", name)
        })
    }

    suspend fun releaseSubdomain(id: Long) {
        request<JsonElement>("/subdomains/$id", "DELETE")
    }

    suspend fun getDnsRecords(subdomainID: Long): List<ClientDnsRecord> =
        request("/subdomains/$subdomainID/records")

    suspend fun createDnsRecord(subdomainID: Long, type: String, content: String, proxied: Boolean) {
        request<JsonElement>("/subdomains/$subdomainID/records", "POST", jsonBody {
            put("type", type)
            put("content", content)
            put("proxied", proxied)
        })
    }

    suspend fun updateDnsRecord(subdomainID: Long, recordID: Long, content: String, proxied: Boolean) {
        request<JsonElement>("/subdomains/$subdomainID/records/$recordID", "PUT", jsonBody {
            put("content", content)
            put("proxied", proxied)
        })
    }

    suspend fun deleteDnsRecord(subdomainID: Long, recordID: Long) {
        request<JsonElement>("/subdomains/$subdomainID/records/$recordID", "DELETE")
    }

    suspend fun getDomains(): List<ClientDomain> = request("/domains")

    suspend fun getCredits(): ClientCreditBalance = request("/credits")

    suspend fun getDailyCheckinStatus(): ClientDailyCheckinStatus = request("/credits/checkin/status")

    suspend fun claimDailyCheckin() {
        request<JsonElement>("/credits/checkin", "POST", jsonBody {})
    }

    suspend fun getNotifications(offset: Int = 0, limit: Int = 20): OffsetPage<List<ClientNotification>> =
        requestOffset("/notifications?offset=$offset&limit=$limit")

    suspend fun markNotificationRead(id: Long) {
        request<JsonElement>("/notifications/$id/read", "POST", jsonBody {})
    }

    suspend fun getFriendLinks(): List<ClientFriendLink> = request("/friend-links")

    suspend fun updateProfile(name: String, avatarUrl: String, bio: String, website: String): ClientUser =
        request<ProfilePayload>("/auth/profile", "PUT", jsonBody {
            put("name", name)
            put("avatar_url", avatarUrl)
            put("bio", bio)
            put("website", website)
        }).user

    private fun jsonBody(content: JsonObjectBuilder.() -> Unit): RequestBody =
        json.encodeToString(buildJsonObject(content)).toRequestBody(JSON_MEDIA_TYPE)

    private suspend inline fun <reified T> request(path: String, method: String = "GET", body: RequestBody? = null): T =
        withContext(Dispatchers.IO) {
            val requestBuilder = Request.Builder().url(ClientBuild.apiBaseUrl.trimEnd('/') + path)
            when (method) {
                "POST" -> requestBuilder.post(requireNotNull(body))
                "PUT" -> requestBuilder.put(requireNotNull(body))
                "DELETE" -> if (body == null) requestBuilder.delete() else requestBuilder.delete(body)
                else -> requestBuilder.get()
            }
            http.newCall(requestBuilder.build()).execute().use { response ->
                val bodyText = response.body?.string().orEmpty()
                if (!response.isSuccessful) throw apiException(response.code, response.message, bodyText)
                val envelope = try {
                    json.decodeFromString<ApiEnvelope<T>>(bodyText)
                } catch (_: Exception) {
                    throw ClientApiException(response.code, "The service returned an invalid response.")
                }
                if (envelope.code != 0) throw ClientApiException(response.code, envelope.message.ifBlank { "The service rejected this request." })
                envelope.data ?: throw ClientApiException(response.code, "The service returned an empty response.")
            }
        }

    private suspend inline fun <reified T> requestOffset(path: String): OffsetPage<T> =
        withContext(Dispatchers.IO) {
            val request = Request.Builder().url(ClientBuild.apiBaseUrl.trimEnd('/') + path).get().build()
            http.newCall(request).execute().use { response ->
                val bodyText = response.body?.string().orEmpty()
                if (!response.isSuccessful) throw apiException(response.code, response.message, bodyText)
                val envelope = try {
                    json.decodeFromString<OffsetApiEnvelope<T>>(bodyText)
                } catch (_: Exception) {
                    throw ClientApiException(response.code, "The service returned an invalid response.")
                }
                if (envelope.code != 0) throw ClientApiException(response.code, envelope.message.ifBlank { "The service rejected this request." })
                OffsetPage(
                    items = envelope.data ?: throw ClientApiException(response.code, "The service returned an empty response."),
                    total = envelope.total,
                    offset = envelope.offset,
                    limit = envelope.limit,
                )
            }
        }

    private fun apiException(statusCode: Int, fallback: String, bodyText: String): ClientApiException {
        val message = runCatching { json.decodeFromString<ApiEnvelope<JsonElement>>(bodyText).message }
            .getOrNull()
            .orEmpty()
            .ifBlank { fallback }
        return ClientApiException(statusCode, message)
    }

    private companion object {
        val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()
        val SAFE_METHODS = setOf("GET", "HEAD", "OPTIONS")
    }
}
