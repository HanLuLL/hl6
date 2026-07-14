package cloud.houlang.hl6.client

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.activity.viewModels

class MainActivity : ComponentActivity() {
    private val viewModel: MainViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        NativeOidc.codeFromRedirect(intent?.data)?.let(viewModel::handleNativeOidcCode)
        setContent {
            Hl6Theme(darkTheme = isSystemInDarkTheme()) {
                Hl6NativeApp(viewModel)
            }
        }
    }

    override fun onNewIntent(intent: android.content.Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        NativeOidc.codeFromRedirect(intent.data)?.let(viewModel::handleNativeOidcCode)
    }
}
