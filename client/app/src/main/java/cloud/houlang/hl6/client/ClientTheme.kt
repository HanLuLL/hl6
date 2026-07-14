package cloud.houlang.hl6.client

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.cos
import kotlin.math.pow
import kotlin.math.sin

object Hl6Tokens {
    val Radius = 10.dp
    val CardRadius = RoundedCornerShape(Radius)
    val Brand = oklch(0.548, 0.219, 264.0)
    val BrandHover = oklch(0.478, 0.205, 264.0)
    val Success = oklch(0.62, 0.14, 145.0)
    val Warning = oklch(0.70, 0.14, 75.0)
    val Danger = oklch(0.577, 0.245, 27.325)
}

private val LightScheme: ColorScheme = lightColorScheme(
    primary = Hl6Tokens.Brand,
    onPrimary = oklch(0.99, 0.0, 0.0),
    secondary = oklch(0.965, 0.02, 264.0),
    onSecondary = oklch(0.32, 0.08, 264.0),
    background = oklch(0.995, 0.006, 264.0),
    onBackground = oklch(0.195, 0.025, 264.0),
    surface = oklch(1.0, 0.004, 264.0),
    onSurface = oklch(0.195, 0.025, 264.0),
    surfaceVariant = oklch(0.965, 0.015, 264.0),
    onSurfaceVariant = oklch(0.5, 0.04, 264.0),
    outline = oklch(0.905, 0.022, 264.0),
    error = Hl6Tokens.Danger,
    onError = Color.White,
)

private val DarkScheme: ColorScheme = darkColorScheme(
    primary = oklch(0.62, 0.19, 264.0),
    onPrimary = oklch(0.99, 0.0, 0.0),
    secondary = oklch(0.26, 0.035, 264.0),
    onSecondary = oklch(0.92, 0.01, 264.0),
    background = oklch(0.155, 0.025, 264.0),
    onBackground = oklch(0.97, 0.008, 264.0),
    surface = oklch(0.205, 0.03, 264.0),
    onSurface = oklch(0.97, 0.008, 264.0),
    surfaceVariant = oklch(0.26, 0.03, 264.0),
    onSurfaceVariant = oklch(0.68, 0.04, 264.0),
    outline = oklch(0.35, 0.03, 264.0),
    error = oklch(0.704, 0.191, 22.216),
    onError = oklch(0.155, 0.025, 264.0),
)

private val Hl6Typography = Typography(
    displaySmall = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 30.sp, lineHeight = 36.sp, letterSpacing = 0.sp),
    headlineSmall = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 24.sp, lineHeight = 29.sp, letterSpacing = 0.sp),
    titleLarge = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 20.sp, lineHeight = 25.sp, letterSpacing = 0.sp),
    titleMedium = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 16.sp, lineHeight = 21.sp, letterSpacing = 0.sp),
    bodyLarge = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 16.sp, lineHeight = 24.sp, letterSpacing = 0.sp),
    bodyMedium = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 14.sp, lineHeight = 20.sp, letterSpacing = 0.sp),
    labelMedium = TextStyle(fontFamily = FontFamily.SansSerif, fontSize = 12.sp, lineHeight = 16.sp, letterSpacing = 0.sp),
)

@Composable
fun Hl6Theme(darkTheme: Boolean, content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkScheme else LightScheme,
        typography = Hl6Typography,
        shapes = MaterialTheme.shapes.copy(
            extraSmall = RoundedCornerShape(6.dp),
            small = RoundedCornerShape(8.dp),
            medium = Hl6Tokens.CardRadius,
            large = RoundedCornerShape(14.dp),
        ),
        content = content,
    )
}

// CSS uses OKLCH. Converting it here keeps native colors aligned with web/src/index.css.
private fun oklch(lightness: Double, chroma: Double, hue: Double): Color {
    val radians = hue * Math.PI / 180.0
    val a = chroma * cos(radians)
    val b = chroma * sin(radians)
    val l = lightness + 0.3963377774 * a + 0.2158037573 * b
    val m = lightness - 0.1055613458 * a - 0.0638541728 * b
    val s = lightness - 0.0894841775 * a - 1.2914855480 * b
    val l3 = l * l * l
    val m3 = m * m * m
    val s3 = s * s * s
    val red = 4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3
    val green = -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3
    val blue = -0.0041960863 * l3 - 0.7034186147 * m3 + 1.7076147010 * s3
    return Color(toSrgb(red), toSrgb(green), toSrgb(blue))
}

private fun toSrgb(value: Double): Float {
    val encoded = if (value <= 0.0031308) 12.92 * value else 1.055 * value.pow(1.0 / 2.4) - 0.055
    return encoded.coerceIn(0.0, 1.0).toFloat()
}
