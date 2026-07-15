package cloud.houlang.hl6.client

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountCircle
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.ArrowBack
import androidx.compose.material.icons.outlined.CreditCard
import androidx.compose.material.icons.outlined.Dashboard
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Dns
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.Language
import androidx.compose.material.icons.outlined.Logout
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material.icons.outlined.Security
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationRail
import androidx.compose.material3.NavigationRailItem
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.VerticalDivider
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.flow.collect

private enum class ClientDestination(val label: String, val icon: ImageVector) {
    Dashboard("概览", Icons.Outlined.Dashboard),
    Domains("域名", Icons.Outlined.Language),
    Subdomains("子域名", Icons.Outlined.Dns),
    Credits("积分", Icons.Outlined.CreditCard),
    Profile("资料", Icons.Outlined.AccountCircle),
    FriendLinks("友链", Icons.Outlined.Language),
}

@Composable
fun Hl6NativeApp(viewModel: MainViewModel) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val context = LocalContext.current
    when (val current = state) {
        ClientState.Loading -> LoadingScreen()
        ClientState.SignedOut -> SignInScreen {
            viewModel.startNativeOidc { loginURL -> NativeOidc.launch(context, loginURL) }
        }
        ClientState.CommunicationKeyInvalid -> CommunicationKeyScreen()
        is ClientState.Ready -> AuthenticatedApp(current, viewModel)
        is ClientState.Banned -> BannedScreen(current.user, viewModel::signOut)
        is ClientState.Error -> ErrorScreen(current.message, viewModel::refresh)
    }
}

@Composable
private fun LoadingScreen() {
    Surface(modifier = Modifier.fillMaxSize()) {
        Box(contentAlignment = Alignment.Center, modifier = Modifier.fillMaxSize()) {
            CircularProgressIndicator(color = Hl6Tokens.Brand)
        }
    }
}

@Composable
private fun SignInScreen(onSignIn: () -> Unit) {
    Surface(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.fillMaxSize().padding(28.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                modifier = Modifier.size(68.dp).clip(Hl6Tokens.CardRadius).background(Hl6Tokens.Brand),
                contentAlignment = Alignment.Center,
            ) {
                Text(ClientBuild.displayName.take(2).uppercase(), color = MaterialTheme.colorScheme.onPrimary, fontWeight = FontWeight.Bold)
            }
            Spacer(Modifier.height(24.dp))
            Text(ClientBuild.displayName, style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(8.dp))
            Text("使用统一账号登录以管理你的域名和 DNS 记录。", color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(28.dp))
            Button(onClick = onSignIn, modifier = Modifier.fillMaxWidth()) {
                Text("使用 OIDC 登录")
            }
        }
    }
}

@Composable
private fun CommunicationKeyScreen() {
    Surface(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.fillMaxSize().padding(28.dp),
            verticalArrangement = Arrangement.Center,
        ) {
            Icon(Icons.Outlined.Security, contentDescription = null, tint = MaterialTheme.colorScheme.error, modifier = Modifier.size(38.dp))
            Spacer(Modifier.height(16.dp))
            Text("客户端通讯密钥不可用", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(8.dp))
            Text("网站已作废或更新通讯密钥。请安装由管理员重新构建的客户端后再继续使用。", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun ErrorScreen(message: String, retry: () -> Unit) {
    Surface(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.fillMaxSize().padding(28.dp),
            verticalArrangement = Arrangement.Center,
        ) {
            Text("无法连接", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(8.dp))
            Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(20.dp))
            OutlinedButton(onClick = retry) {
                Icon(Icons.Outlined.Refresh, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text("重试")
            }
        }
    }
}

@Composable
private fun BannedScreen(user: ClientUser, signOut: () -> Unit) {
    Surface(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.fillMaxSize().padding(28.dp),
            verticalArrangement = Arrangement.Center,
        ) {
            Icon(Icons.Outlined.Security, contentDescription = null, tint = MaterialTheme.colorScheme.error, modifier = Modifier.size(38.dp))
            Spacer(Modifier.height(16.dp))
            Text("账号已被封禁", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(16.dp))
            NativeCard {
                Text("封禁原因", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text(user.banned_reason.ifBlank { "-" }, modifier = Modifier.padding(top = 4.dp))
                user.banned_at?.let { Text("封禁时间：$it", style = MaterialTheme.typography.labelMedium, modifier = Modifier.padding(top = 12.dp)) }
                Text(
                    user.banned_until?.let { "预计解封时间：$it" } ?: "预计解封时间：需管理员审核后解除",
                    style = MaterialTheme.typography.labelMedium,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
            Spacer(Modifier.height(20.dp))
            OutlinedButton(onClick = signOut) {
                Icon(Icons.Outlined.Logout, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text("安全退出")
            }
        }
    }
}

@Composable
private fun AuthenticatedApp(state: ClientState.Ready, viewModel: MainViewModel) {
    var destination by rememberSaveable { mutableStateOf(ClientDestination.Dashboard) }
    var updateVisible by remember(state.update) { mutableStateOf(state.update != null) }
    var notificationsVisible by rememberSaveable { mutableStateOf(false) }
    val context = LocalContext.current
    val detailState by viewModel.subdomainDetail.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(viewModel) {
        viewModel.messages.collect { snackbarHostState.showSnackbar(it) }
    }

    if (state.update != null && updateVisible) {
        AlertDialog(
            onDismissRequest = { if (!state.update.force_update) updateVisible = false },
            title = { Text("客户端有可用更新") },
            text = {
                Text(buildString {
                    append("最新版本：${state.update.latest_version}")
                    if (state.update.update_notice.isNotBlank()) append("\n\n${state.update.update_notice}")
                })
            },
            dismissButton = if (!state.update.force_update) {
                { OutlinedButton(onClick = { updateVisible = false }) { Text("稍后更新") } }
            } else null,
            confirmButton = {
                Button(
                    onClick = {
                        state.update.update_url.takeIf { it.isNotBlank() }?.let {
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(it)))
                        }
                    },
                    enabled = state.update.update_url.isNotBlank(),
                ) { Text("立即更新") }
            },
        )
    }

    if (notificationsVisible) {
        NotificationsDialog(
            notifications = state.notifications,
            onDismiss = { notificationsVisible = false },
            onRead = viewModel::markNotificationRead,
        )
    }

    BoxWithConstraints(modifier = Modifier.fillMaxSize()) {
        val compactLayout = maxWidth < 720.dp
        if (compactLayout) {
            Scaffold(
                topBar = {
                    NativeTopBar(
                        state = state,
                        onShowNotifications = { notificationsVisible = true },
                    )
                },
                bottomBar = {
                    NavigationBar {
                        ClientDestination.entries.forEach { item ->
                            NavigationBarItem(
                                selected = destination == item,
                                onClick = { destination = item },
                                icon = { Icon(item.icon, contentDescription = item.label) },
                                label = { Text(item.label) },
                            )
                        }
                    }
                },
                snackbarHost = { SnackbarHost(snackbarHostState) },
            ) { padding ->
                NativeDestinationContent(state, destination, detailState, viewModel, Modifier.padding(padding))
            }
        } else {
            Row(modifier = Modifier.fillMaxSize()) {
                NavigationRail(modifier = Modifier.fillMaxHeight()) {
                    ClientDestination.entries.forEach { item ->
                        NavigationRailItem(
                            selected = destination == item,
                            onClick = { destination = item },
                            icon = { Icon(item.icon, contentDescription = item.label) },
                            label = { Text(item.label) },
                        )
                    }
                }
                VerticalDivider(modifier = Modifier.fillMaxHeight().width(1.dp))
                Column(modifier = Modifier.weight(1f)) {
                    NativeTopBar(
                        state = state,
                        onShowNotifications = { notificationsVisible = true },
                    )
                    Scaffold(snackbarHost = { SnackbarHost(snackbarHostState) }) { padding ->
                        NativeDestinationContent(state, destination, detailState, viewModel, Modifier.padding(padding))
                    }
                }
            }
        }
    }
}

@Composable
private fun NativeDestinationContent(
    state: ClientState.Ready,
    destination: ClientDestination,
    detailState: SubdomainDetailState,
    viewModel: MainViewModel,
    modifier: Modifier,
) {
    Box(modifier = modifier.fillMaxSize()) {
        if (detailState !is SubdomainDetailState.Idle) {
            SubdomainDetailScreen(detailState, viewModel)
        } else {
            when (destination) {
                ClientDestination.Dashboard -> DashboardScreen(state.user, state.credits, state.subdomains, viewModel::loadSubdomainDetail)
                ClientDestination.Domains -> DomainsScreen(state.domains, viewModel::claimSubdomain)
                ClientDestination.Subdomains -> SubdomainsScreen(state.subdomains, viewModel::loadSubdomainDetail)
                ClientDestination.Credits -> CreditsScreen(state.creditBalance, state.dailyCheckin, viewModel::claimDailyCheckin)
                ClientDestination.Profile -> ProfileScreen(state.user, viewModel::updateProfile, viewModel::signOut)
                ClientDestination.FriendLinks -> FriendLinksScreen(state.friendLinks)
            }
        }
    }
}

@Composable
private fun DashboardScreen(
    user: ClientUser,
    credits: Double,
    subdomains: List<ClientSubdomain>,
    openSubdomain: (Long) -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Avatar(user)
                Spacer(Modifier.width(12.dp))
                Column {
                    Text("控制台", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                    Text("欢迎回来，${user.name.ifBlank { user.email }}", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
                StatCard("积分余额", credits.toString(), Modifier.weight(1f))
                StatCard("我的子域名", subdomains.size.toString(), Modifier.weight(1f))
            }
        }
        item {
            Text("我的子域名", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        }
        if (subdomains.isEmpty()) {
            item { EmptyState("暂无已认领子域名") }
        } else {
            items(subdomains, key = { it.id }) { subdomain ->
                NativeCard(modifier = Modifier.clickable { openSubdomain(subdomain.id) }) {
                    Text(subdomain.fqdn.ifBlank { subdomain.name }, fontWeight = FontWeight.SemiBold)
                    Text(subdomain.status, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
}

@Composable
private fun SubdomainsScreen(subdomains: List<ClientSubdomain>, openSubdomain: (Long) -> Unit) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item { Text("我的子域名", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold) }
        if (subdomains.isEmpty()) item { EmptyState("暂无已认领子域名") }
        items(subdomains, key = { it.id }) { subdomain ->
            NativeCard(modifier = Modifier.clickable { openSubdomain(subdomain.id) }) {
                Text(subdomain.fqdn.ifBlank { subdomain.name }, fontWeight = FontWeight.SemiBold)
                Text("状态：${subdomain.status}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun DomainsScreen(domains: List<ClientDomain>, claimSubdomain: (Long, String) -> Unit) {
    var selectedDomainID by rememberSaveable { mutableStateOf<Long?>(null) }
    val selectedDomain = domains.firstOrNull { it.id == selectedDomainID }
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Column {
                Text("域名", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                Text("在可用域名下认领子域名。", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        if (domains.isEmpty()) item { EmptyState("当前没有可认领域名") }
        items(domains, key = { it.id }) { domain ->
            NativeCard {
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(domain.name, fontWeight = FontWeight.SemiBold)
                        if (domain.description.isNotBlank()) {
                            Text(domain.description, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    Text("${domain.credit_cost} 积分", style = MaterialTheme.typography.labelMedium, color = Hl6Tokens.Brand)
                }
                TextButton(
                    onClick = { selectedDomainID = domain.id },
                    modifier = Modifier.padding(top = 8.dp),
                ) { Text("认领子域名") }
            }
        }
    }
    if (selectedDomain != null) {
        ClaimSubdomainDialog(
            domain = selectedDomain,
            onDismiss = { selectedDomainID = null },
            onConfirm = { name ->
                claimSubdomain(selectedDomain.id, name)
                selectedDomainID = null
            },
        )
    }
}

@Composable
private fun CreditsScreen(
    balance: ClientCreditBalance,
    checkin: ClientDailyCheckinStatus,
    claimDailyCheckin: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item { Text("积分", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold) }
        item {
            NativeCard {
                Text("当前余额", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text(balance.balance.toString(), style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
            }
        }
        if (checkin.enabled) {
            item {
                NativeCard {
                    Text("每日签到", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                    Text(
                        if (checkin.claimed_today) "今日已签到" else "签到可获得 ${checkin.reward} 积分",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                    Button(
                        onClick = claimDailyCheckin,
                        enabled = !checkin.claimed_today,
                        modifier = Modifier.padding(top = 12.dp),
                    ) { Text(if (checkin.claimed_today) "已签到" else "立即签到") }
                }
            }
        }
        item { Text("最近交易", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold) }
        if (balance.transactions.isEmpty()) item { EmptyState("暂无积分交易") }
        items(balance.transactions, key = { it.id }) { transaction ->
            NativeCard {
                Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(transaction.description_key.ifBlank { "积分交易" }, fontWeight = FontWeight.Medium)
                        Text(transaction.created_at, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Text(
                        if (transaction.amount > 0) "+${transaction.amount}" else transaction.amount.toString(),
                        color = if (transaction.amount >= 0) Hl6Tokens.Success else MaterialTheme.colorScheme.error,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }
        }
    }
}

@Composable
private fun FriendLinksScreen(links: List<ClientFriendLink>) {
    val context = LocalContext.current
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Column {
                Text("友情链接", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                Text("发现相关服务与社区。", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        if (links.isEmpty()) item { EmptyState("暂无友情链接") }
        items(links, key = { it.id }) { link ->
            NativeCard {
                Column(
                    modifier = Modifier.fillMaxWidth().clickable {
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(link.url)))
                    },
                ) {
                    Text(link.name, fontWeight = FontWeight.SemiBold)
                    if (link.description.isNotBlank()) {
                        Text(link.description, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Text(link.url, style = MaterialTheme.typography.labelMedium, color = Hl6Tokens.Brand, modifier = Modifier.padding(top = 6.dp))
                }
            }
        }
    }
}

@Composable
private fun ProfileScreen(
    user: ClientUser,
    saveProfile: (String, String, String, String) -> Unit,
    signOut: () -> Unit,
) {
    var name by remember(user.id) { mutableStateOf(user.name) }
    var avatarUrl by remember(user.id) { mutableStateOf(user.avatar_url) }
    var bio by remember(user.id) { mutableStateOf(user.bio) }
    var website by remember(user.id) { mutableStateOf(user.website) }
    Column(
        modifier = Modifier.fillMaxSize().padding(20.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("个人资料", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
        NativeCard {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Avatar(user)
                Spacer(Modifier.width(12.dp))
                Column {
                    Text(user.name.ifBlank { "User" }, fontWeight = FontWeight.SemiBold)
                    Text(user.email, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("姓名") }, singleLine = true, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = avatarUrl, onValueChange = { avatarUrl = it }, label = { Text("头像 URL") }, singleLine = true, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = website, onValueChange = { website = it }, label = { Text("网站") }, singleLine = true, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = bio, onValueChange = { bio = it }, label = { Text("简介") }, modifier = Modifier.fillMaxWidth(), minLines = 3)
        Button(onClick = { saveProfile(name, avatarUrl, bio, website) }, modifier = Modifier.fillMaxWidth()) { Text("保存资料") }
        OutlinedButton(onClick = signOut) {
            Icon(Icons.Outlined.Logout, contentDescription = null)
            Spacer(Modifier.width(8.dp))
            Text("退出登录")
        }
    }
}

@Composable
private fun NativeTopBar(state: ClientState.Ready, onShowNotifications: () -> Unit) {
    val unreadCount = state.notifications.count { !it.is_read }
    Surface(modifier = Modifier.fillMaxWidth(), color = MaterialTheme.colorScheme.background) {
        Row(
            modifier = Modifier.fillMaxWidth().height(56.dp).padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(ClientBuild.displayName, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            Spacer(Modifier.weight(1f))
            Text("${state.credits}", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.width(8.dp))
            IconButton(onClick = onShowNotifications) {
                BadgedBox(
                    badge = {
                        if (unreadCount > 0) {
                            Badge { Text(if (unreadCount > 9) "9+" else unreadCount.toString()) }
                        }
                    },
                ) {
                    Icon(Icons.Outlined.Notifications, contentDescription = "Notifications")
                }
            }
            Avatar(state.user)
        }
    }
}

@Composable
private fun NotificationsDialog(
    notifications: List<ClientNotification>,
    onDismiss: () -> Unit,
    onRead: (Long) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("通知") },
        text = {
            if (notifications.isEmpty()) {
                Text("暂无通知", color = MaterialTheme.colorScheme.onSurfaceVariant)
            } else {
                LazyColumn(
                    modifier = Modifier.height(320.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    items(notifications, key = { it.id }) { notification ->
                        Column(
                            modifier = Modifier.fillMaxWidth().clickable {
                                if (!notification.is_read) onRead(notification.id)
                            },
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(
                                    notification.title,
                                    fontWeight = if (notification.is_read) FontWeight.Medium else FontWeight.Bold,
                                    modifier = Modifier.weight(1f),
                                )
                                if (!notification.is_read) {
                                    Badge { Text("新") }
                                }
                            }
                            if (notification.content.isNotBlank()) {
                                Text(notification.content, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Text(notification.created_at, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text("关闭") } },
    )
}

@Composable
private fun ClaimSubdomainDialog(
    domain: ClientDomain,
    onDismiss: () -> Unit,
    onConfirm: (String) -> Unit,
) {
    var name by remember(domain.id) { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("认领 ${domain.name}") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("${domain.credit_cost} 积分", color = MaterialTheme.colorScheme.onSurfaceVariant)
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("子域名前缀") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("取消") } },
        confirmButton = { Button(onClick = { onConfirm(name) }) { Text("认领") } },
    )
}

@Composable
private fun SubdomainDetailScreen(detailState: SubdomainDetailState, viewModel: MainViewModel) {
    when (detailState) {
        SubdomainDetailState.Idle -> Unit
        SubdomainDetailState.Loading -> LoadingScreen()
        is SubdomainDetailState.Error -> {
            Column(modifier = Modifier.fillMaxSize().padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                IconButton(onClick = viewModel::closeSubdomainDetail) {
                    Icon(Icons.Outlined.ArrowBack, contentDescription = "Back")
                }
                Text(detailState.message, color = MaterialTheme.colorScheme.error)
            }
        }
        is SubdomainDetailState.Ready -> SubdomainDetailContent(detailState, viewModel)
    }
}

@Composable
private fun SubdomainDetailContent(detailState: SubdomainDetailState.Ready, viewModel: MainViewModel) {
    val subdomain = detailState.subdomain
    val readOnly = subdomain.status != "active" || subdomain.domain?.migration_read_only == true
    var editorOpen by rememberSaveable { mutableStateOf(false) }
    var editingRecordID by rememberSaveable { mutableStateOf<Long?>(null) }
    var deletingRecordID by rememberSaveable { mutableStateOf<Long?>(null) }
    var releaseVisible by rememberSaveable { mutableStateOf(false) }
    val editingRecord = detailState.records.firstOrNull { it.id == editingRecordID }
    val deletingRecord = detailState.records.firstOrNull { it.id == deletingRecordID }

    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                IconButton(onClick = viewModel::closeSubdomainDetail) {
                    Icon(Icons.Outlined.ArrowBack, contentDescription = "Back")
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(subdomain.fqdn.ifBlank { subdomain.name }, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                    Text(subdomain.domain?.name.orEmpty(), color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        if (subdomain.status != "active") {
            item {
                NativeCard {
                    Text("子域名已暂停", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.SemiBold)
                    if (subdomain.suspended_reason.isNotBlank()) Text(subdomain.suspended_reason)
                }
            }
        }
        if (subdomain.domain?.migration_read_only == true) {
            item {
                NativeCard { Text("域名迁移期间暂时只读", color = MaterialTheme.colorScheme.error) }
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = { editingRecordID = null; editorOpen = true }, enabled = !readOnly) {
                    Icon(Icons.Outlined.Add, contentDescription = null)
                    Spacer(Modifier.width(6.dp))
                    Text("添加记录")
                }
                OutlinedButton(onClick = { releaseVisible = true }) {
                    Icon(Icons.Outlined.Delete, contentDescription = null)
                    Spacer(Modifier.width(6.dp))
                    Text("释放")
                }
            }
        }
        item { Text("DNS 记录", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold) }
        if (detailState.records.isEmpty()) {
            item { EmptyState("暂无 DNS 记录") }
        }
        items(detailState.records, key = { it.id }) { record ->
            NativeCard {
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(record.type, fontWeight = FontWeight.SemiBold)
                        Text(record.content, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        Text(
                            if (record.proxied) "已代理" else "仅 DNS",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    IconButton(onClick = { editingRecordID = record.id; editorOpen = true }, enabled = !readOnly) {
                        Icon(Icons.Outlined.Edit, contentDescription = "Edit")
                    }
                    IconButton(onClick = { deletingRecordID = record.id }, enabled = !readOnly) {
                        Icon(Icons.Outlined.Delete, contentDescription = "Delete")
                    }
                }
            }
        }
    }

    if (editorOpen) {
        DnsRecordEditorDialog(
            record = editingRecord,
            onDismiss = { editorOpen = false },
            onSave = { type, content, proxied ->
                if (editingRecord == null) {
                    viewModel.createDnsRecord(subdomain.id, type, content, proxied)
                } else {
                    viewModel.updateDnsRecord(subdomain.id, editingRecord.id, content, proxied)
                }
                editorOpen = false
            },
        )
    }
    if (deletingRecord != null) {
        AlertDialog(
            onDismissRequest = { deletingRecordID = null },
            title = { Text("删除 DNS 记录") },
            text = { Text("将删除 ${deletingRecord.type} 记录：${deletingRecord.content}") },
            dismissButton = { TextButton(onClick = { deletingRecordID = null }) { Text("取消") } },
            confirmButton = {
                Button(
                    onClick = {
                        viewModel.deleteDnsRecord(subdomain.id, deletingRecord.id)
                        deletingRecordID = null
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                ) { Text("删除") }
            },
        )
    }
    if (releaseVisible) {
        AlertDialog(
            onDismissRequest = { releaseVisible = false },
            title = { Text("释放子域名") },
            text = { Text("确认释放 ${subdomain.fqdn.ifBlank { subdomain.name }} 吗？此操作由服务端执行且不可撤销。") },
            dismissButton = { TextButton(onClick = { releaseVisible = false }) { Text("取消") } },
            confirmButton = {
                Button(
                    onClick = {
                        viewModel.releaseSubdomain(subdomain.id)
                        releaseVisible = false
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                ) { Text("释放") }
            },
        )
    }
}

@Composable
private fun DnsRecordEditorDialog(
    record: ClientDnsRecord?,
    onDismiss: () -> Unit,
    onSave: (String, String, Boolean) -> Unit,
) {
    var type by remember(record?.id) { mutableStateOf(record?.type ?: "A") }
    var content by remember(record?.id) { mutableStateOf(record?.content ?: "") }
    var proxied by remember(record?.id) { mutableStateOf(record?.proxied ?: false) }
    var typeMenuOpen by remember { mutableStateOf(false) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(if (record == null) "添加 DNS 记录" else "编辑 DNS 记录") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                if (record == null) {
                    Box {
                        OutlinedButton(onClick = { typeMenuOpen = true }, modifier = Modifier.fillMaxWidth()) { Text(type) }
                        DropdownMenu(expanded = typeMenuOpen, onDismissRequest = { typeMenuOpen = false }) {
                            listOf("A", "AAAA", "CNAME", "TXT").forEach { option ->
                                DropdownMenuItem(
                                    text = { Text(option) },
                                    onClick = { type = option; typeMenuOpen = false },
                                )
                            }
                        }
                    }
                } else {
                    Text("记录类型：${record.type}", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                OutlinedTextField(
                    value = content,
                    onValueChange = { content = it },
                    label = { Text("记录内容") },
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("Cloudflare 代理", modifier = Modifier.weight(1f))
                    Switch(checked = proxied, onCheckedChange = { proxied = it })
                }
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("取消") } },
        confirmButton = { Button(onClick = { onSave(type, content, proxied) }) { Text("保存") } },
    )
}

@Composable
private fun NativeCard(
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        shape = Hl6Tokens.CardRadius,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
    ) {
        Column(modifier = Modifier.padding(16.dp), content = content)
    }
}

@Composable
private fun StatCard(title: String, value: String, modifier: Modifier = Modifier) {
    Card(
        modifier = modifier,
        shape = Hl6Tokens.CardRadius,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(title, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(8.dp))
            Text(value, style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
private fun EmptyState(message: String) {
    NativeCard {
        Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun Avatar(user: ClientUser) {
    Box(
        modifier = Modifier.size(44.dp).clip(CircleShape).background(MaterialTheme.colorScheme.secondary),
        contentAlignment = Alignment.Center,
    ) {
        Text(user.name.firstOrNull()?.uppercase() ?: "U", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.onSecondary)
    }
}
