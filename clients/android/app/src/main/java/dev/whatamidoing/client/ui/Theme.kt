package dev.whatamidoing.client.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

// 与服务端网页 m3.css 同一套 M3 色板，叠加 MIUI X 视觉语言。

private val LightColors = lightColorScheme(
    primary = Color(0xFF5B5BD6),
    onPrimary = Color.White,
    primaryContainer = Color(0xFFE0E0FF),
    onPrimaryContainer = Color(0xFF00006E),
    secondaryContainer = Color(0xFFE3E1F9),
    onSecondaryContainer = Color(0xFF1A1B2E),
    tertiary = Color(0xFF006D4B),
    background = Color(0xFFFBF8FF),
    surface = Color(0xFFFBF8FF),
    surfaceContainer = Color(0xFFEFECF4),
    surfaceContainerHigh = Color(0xFFE9E6EE),
    onSurface = Color(0xFF1C1B20),
    onSurfaceVariant = Color(0xFF48464E),
    outline = Color(0xFF79767F),
    error = Color(0xFFBA1A1A),
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFFBFBFFF),
    onPrimary = Color(0xFF1A1A86),
    primaryContainer = Color(0xFF4242BD),
    onPrimaryContainer = Color(0xFFE0E0FF),
    secondaryContainer = Color(0xFF353645),
    onSecondaryContainer = Color(0xFFE3E1F9),
    tertiary = Color(0xFF68DBAB),
    background = Color(0xFF141318),
    surface = Color(0xFF141318),
    surfaceContainer = Color(0xFF201F24),
    surfaceContainerHigh = Color(0xFF2A2930),
    onSurface = Color(0xFFE4E1E9),
    onSurfaceVariant = Color(0xFFCAC6D0),
    outline = Color(0xFF948F99),
    error = Color(0xFFFFB4AB),
)

@Composable
fun AppTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    MaterialTheme(
        colorScheme = if (dark) DarkColors else LightColors,
        // MIUI X：标题更粗、卡片圆角更大
        typography = Typography().run {
            copy(
                headlineMedium = headlineMedium.copy(fontWeight = FontWeight.Bold),
                titleLarge = titleLarge.copy(fontWeight = FontWeight.SemiBold),
                titleMedium = titleMedium.copy(fontWeight = FontWeight.Medium),
            )
        },
        shapes = Shapes(
            extraSmall = RoundedCornerShape(8.dp),
            small = RoundedCornerShape(12.dp),
            medium = RoundedCornerShape(20.dp),
            large = RoundedCornerShape(24.dp),
            extraLarge = RoundedCornerShape(32.dp),
        ),
        content = content,
    )
}
