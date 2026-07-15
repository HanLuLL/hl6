import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.kotlin.plugin.serialization")
}

val clientProperties = Properties().apply {
    val file = rootProject.file("local.properties")
    check(file.isFile) { "client/local.properties is required. Run client/scripts/configure-build.mjs before building." }
    file.reader(Charsets.UTF_8).use(::load)
}

fun clientProperty(name: String): String =
    clientProperties.getProperty(name)?.trim()?.takeIf { it.isNotEmpty() }
        ?: error("Missing required client build property: $name")

fun buildConfigString(value: String): String =
    "\"" + value.replace("\\", "\\\\").replace("\"", "\\\"") + "\""

val clientApplicationId = clientProperty("client.applicationId")
val clientVersionCode = clientProperty("client.versionCode").toInt()
val clientVersionName = clientProperty("client.versionName")
val clientDisplayName = clientProperty("client.displayName")
val clientApiBaseUrl = clientProperty("client.apiBaseUrl")
val clientCommunicationKey = clientProperty("client.communicationKey")
val clientNativeRedirectUri = clientProperty("client.nativeRedirectUri")
val clientKeystoreFile = clientProperty("client.keystoreFile")
val clientKeystorePassword = clientProperty("client.keystorePassword")
val clientKeystoreType = clientProperty("client.keystoreType")
val clientKeyAlias = clientProperty("client.keyAlias")
val clientKeyPassword = clientProperty("client.keyPassword")

android {
    namespace = "cloud.houlang.hl6.client"
    compileSdk = 35

    defaultConfig {
        applicationId = clientApplicationId
        minSdk = 23
        targetSdk = 35
        versionCode = clientVersionCode
        versionName = clientVersionName
        manifestPlaceholders["clientAppName"] = clientDisplayName
        manifestPlaceholders["nativeRedirectScheme"] = clientNativeRedirectUri.substringBefore("://")

        buildConfigField("String", "API_BASE_URL", buildConfigString(clientApiBaseUrl))
        buildConfigField("String", "COMMUNICATION_KEY", buildConfigString(clientCommunicationKey))
        buildConfigField("String", "NATIVE_REDIRECT_URI", buildConfigString(clientNativeRedirectUri))
        buildConfigField("String", "CLIENT_VERSION", buildConfigString(clientVersionName))
        buildConfigField("String", "CLIENT_DISPLAY_NAME", buildConfigString(clientDisplayName))
    }

    buildFeatures {
        buildConfig = true
        compose = true
    }

    signingConfigs {
        create("release") {
            storeFile = rootProject.file(clientKeystoreFile)
            storePassword = clientKeystorePassword
            storeType = clientKeystoreType
            keyAlias = clientKeyAlias
            keyPassword = clientKeyPassword
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            signingConfig = signingConfigs.getByName("release")
        }
    }

    packaging {
        resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"
    }
}

dependencies {
    implementation(platform("androidx.compose:compose-bom:2024.12.01"))
    implementation("androidx.activity:activity-compose:1.10.0")
    implementation("androidx.browser:browser:1.8.0")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.navigation:navigation-compose:2.8.5")
    implementation("androidx.security:security-crypto:1.1.0-alpha06")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")

    debugImplementation("androidx.compose.ui:ui-tooling")
}
