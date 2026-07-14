package cloud.houlang.hl6.client

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent

object NativeOidc {
    fun launch(context: Context, loginURL: String) {
        val loginUri = Uri.parse(loginURL)
        require(loginUri.scheme == "https") { "Native OIDC login URL must use HTTPS." }
        CustomTabsIntent.Builder().build().launchUrl(context, loginUri)
    }

    fun codeFromRedirect(uri: Uri?): String? {
        val expected = Uri.parse(ClientBuild.nativeRedirectUri)
        if (
            uri == null ||
            uri.scheme != expected.scheme ||
            uri.host != expected.host ||
            uri.path != expected.path
        ) return null
        return uri.getQueryParameter("code")?.takeIf { it.isNotBlank() }
    }
}
