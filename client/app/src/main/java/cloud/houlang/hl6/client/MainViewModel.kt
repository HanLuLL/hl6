package cloud.houlang.hl6.client

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed interface ClientState {
    data object Loading : ClientState
    data object SignedOut : ClientState
    data object CommunicationKeyInvalid : ClientState
    data class Ready(
        val user: ClientUser,
        val credits: Double,
        val subdomains: List<ClientSubdomain>,
        val domains: List<ClientDomain>,
        val creditBalance: ClientCreditBalance,
        val dailyCheckin: ClientDailyCheckinStatus,
        val friendLinks: List<ClientFriendLink>,
        val notifications: List<ClientNotification>,
        val notificationTotal: Long,
        val update: ClientVersion?,
    ) : ClientState
    data class Banned(val user: ClientUser) : ClientState
    data class Error(val message: String) : ClientState
}

sealed interface SubdomainDetailState {
    data object Idle : SubdomainDetailState
    data object Loading : SubdomainDetailState
    data class Ready(val subdomain: ClientSubdomain, val records: List<ClientDnsRecord>) : SubdomainDetailState
    data class Error(val message: String) : SubdomainDetailState
}

class MainViewModel(application: Application) : AndroidViewModel(application) {
    private val store = SecureClientStore(application)
    private val api = ClientApi(store)
    private val _state = MutableStateFlow<ClientState>(ClientState.Loading)
    private val _subdomainDetail = MutableStateFlow<SubdomainDetailState>(SubdomainDetailState.Idle)
    private val _messages = MutableSharedFlow<String>(extraBufferCapacity = 1)

    val state: StateFlow<ClientState> = _state.asStateFlow()
    val subdomainDetail: StateFlow<SubdomainDetailState> = _subdomainDetail.asStateFlow()
    val messages: SharedFlow<String> = _messages.asSharedFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _state.value = ClientState.Loading
            val version = try {
                api.getVersion()
            } catch (error: ClientApiException) {
                _state.value = if (error.statusCode == 401 || error.statusCode == 403) {
                    ClientState.CommunicationKeyInvalid
                } else {
                    ClientState.Error("Unable to validate the client version.")
                }
                return@launch
            } catch (_: Exception) {
                _state.value = ClientState.Error("Unable to connect to the service.")
                return@launch
            }

            if (store.accessToken().isNullOrBlank()) {
                _state.value = ClientState.SignedOut
                return@launch
            }

            try {
                val me = api.getMe()
                if (me.user.is_banned) {
                    _state.value = ClientState.Banned(me.user)
                    return@launch
                }
                val subdomains = api.getSubdomains()
                val domains = api.getDomains()
                val creditBalance = api.getCredits()
                val dailyCheckin = api.getDailyCheckinStatus()
                val friendLinks = api.getFriendLinks()
                val notifications = api.getNotifications()
                _state.value = ClientState.Ready(
                    user = me.user,
                    credits = me.credits,
                    subdomains = subdomains,
                    domains = domains,
                    creditBalance = creditBalance,
                    dailyCheckin = dailyCheckin,
                    friendLinks = friendLinks,
                    notifications = notifications.items,
                    notificationTotal = notifications.total,
                    update = version.takeIf { it.update_available },
                )
            } catch (error: ClientApiException) {
                if (error.statusCode == 401) {
                    store.clearAccessToken()
                    _state.value = ClientState.SignedOut
                } else if (error.statusCode == 403) {
                    _state.value = ClientState.CommunicationKeyInvalid
                } else {
                    _state.value = ClientState.Error("Unable to load your account.")
                }
            } catch (_: Exception) {
                _state.value = ClientState.Error("Unable to load your account.")
            }
        }
    }

    fun startNativeOidc(openBrowser: (String) -> Unit) {
        viewModelScope.launch {
            try {
                openBrowser(api.startNativeLogin().login_url)
            } catch (error: ClientApiException) {
                if (error.statusCode == 401 || error.statusCode == 403) {
                    _state.value = ClientState.CommunicationKeyInvalid
                } else {
                    emitMessage(error.message.ifBlank { "Unable to start sign-in." })
                }
            } catch (_: Exception) {
                emitMessage("Unable to start sign-in.")
            }
        }
    }

    fun handleNativeOidcCode(code: String) {
        viewModelScope.launch {
            _state.value = ClientState.Loading
            try {
                val exchange = api.exchangeNativeCode(code)
                store.saveAccessToken(exchange.access_token)
                refresh()
            } catch (error: ClientApiException) {
                _state.value = if (error.statusCode == 401 || error.statusCode == 403) {
                    ClientState.CommunicationKeyInvalid
                } else {
                    ClientState.Error("Sign-in could not be completed. Please try again.")
                }
            } catch (_: Exception) {
                _state.value = ClientState.Error("Sign-in could not be completed. Please try again.")
            }
        }
    }

    fun signOut() {
        store.clearAccessToken()
        _subdomainDetail.value = SubdomainDetailState.Idle
        _state.value = ClientState.SignedOut
    }

    fun updateProfile(name: String, avatarUrl: String, bio: String, website: String) {
        val current = _state.value as? ClientState.Ready ?: return
        viewModelScope.launch {
            try {
                val user = api.updateProfile(name, avatarUrl, bio, website)
                _state.value = current.copy(user = user)
                emitMessage("Profile saved.")
            } catch (error: Exception) {
                handleActionError(error, "Unable to save your profile.")
            }
        }
    }

    fun claimSubdomain(domainID: Long, name: String) {
        viewModelScope.launch {
            try {
                api.claimSubdomain(domainID, name)
                emitMessage("Subdomain claimed.")
                refresh()
            } catch (error: Exception) {
                handleActionError(error, "Unable to claim this subdomain.")
            }
        }
    }

    fun releaseSubdomain(id: Long) {
        viewModelScope.launch {
            try {
                api.releaseSubdomain(id)
                _subdomainDetail.value = SubdomainDetailState.Idle
                emitMessage("Subdomain released.")
                refresh()
            } catch (error: Exception) {
                handleActionError(error, "Unable to release this subdomain.")
            }
        }
    }

    fun loadSubdomainDetail(id: Long) {
        viewModelScope.launch {
            _subdomainDetail.value = SubdomainDetailState.Loading
            try {
                val subdomain = api.getSubdomain(id)
                val records = api.getDnsRecords(id)
                _subdomainDetail.value = SubdomainDetailState.Ready(subdomain, records)
            } catch (error: ClientApiException) {
                _subdomainDetail.value = SubdomainDetailState.Error(error.message)
            } catch (_: Exception) {
                _subdomainDetail.value = SubdomainDetailState.Error("Unable to load DNS records.")
            }
        }
    }

    fun closeSubdomainDetail() {
        _subdomainDetail.value = SubdomainDetailState.Idle
    }

    fun createDnsRecord(subdomainID: Long, type: String, content: String, proxied: Boolean) {
        viewModelScope.launch {
            try {
                api.createDnsRecord(subdomainID, type, content, proxied)
                emitMessage("DNS record created.")
                loadSubdomainDetail(subdomainID)
                refresh()
            } catch (error: Exception) {
                handleActionError(error, "Unable to create the DNS record.")
            }
        }
    }

    fun updateDnsRecord(subdomainID: Long, recordID: Long, content: String, proxied: Boolean) {
        viewModelScope.launch {
            try {
                api.updateDnsRecord(subdomainID, recordID, content, proxied)
                emitMessage("DNS record updated.")
                loadSubdomainDetail(subdomainID)
            } catch (error: Exception) {
                handleActionError(error, "Unable to update the DNS record.")
            }
        }
    }

    fun deleteDnsRecord(subdomainID: Long, recordID: Long) {
        viewModelScope.launch {
            try {
                api.deleteDnsRecord(subdomainID, recordID)
                emitMessage("DNS record deleted.")
                loadSubdomainDetail(subdomainID)
                refresh()
            } catch (error: Exception) {
                handleActionError(error, "Unable to delete the DNS record.")
            }
        }
    }

    fun claimDailyCheckin() {
        viewModelScope.launch {
            try {
                api.claimDailyCheckin()
                emitMessage("Daily check-in completed.")
                refresh()
            } catch (error: Exception) {
                handleActionError(error, "Unable to complete daily check-in.")
            }
        }
    }

    fun markNotificationRead(id: Long) {
        val current = _state.value as? ClientState.Ready ?: return
        viewModelScope.launch {
            try {
                api.markNotificationRead(id)
                _state.value = current.copy(
                    notifications = current.notifications.map { notification ->
                        if (notification.id == id) notification.copy(is_read = true) else notification
                    },
                )
            } catch (error: Exception) {
                handleActionError(error, "Unable to mark the notification as read.")
            }
        }
    }

    private fun handleActionError(error: Exception, fallback: String) {
        if (error is ClientApiException && (error.statusCode == 401 || error.statusCode == 403)) {
            refresh()
            return
        }
        emitMessage((error as? ClientApiException)?.message?.ifBlank { fallback } ?: fallback)
    }

    private fun emitMessage(message: String) {
        _messages.tryEmit(message)
    }
}
