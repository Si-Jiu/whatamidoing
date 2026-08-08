plugins {
    // 工具链对齐 miuix 0.9.x（需 compileSdk 37 / AGP 9 / Kotlin 2.4）
    // 注意：AGP 9 已内置 Kotlin，不再需要 org.jetbrains.kotlin.android 插件
    id("com.android.application") version "9.2.1" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.4.0" apply false
}
