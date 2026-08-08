plugins {
    id("com.android.application")
    // AGP 9 内置 Kotlin 支持，无需 org.jetbrains.kotlin.android
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "dev.sijiu49.waid"
    compileSdk = 37

    defaultConfig {
        applicationId = "dev.sijiu49.waid"
        minSdk = 26
        targetSdk = 37
        versionCode = 1
        versionName = "0.1.0"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            // 个人分发：用 debug 签名，保证 CI 产出的 APK 可直接安装。
            // 若上架应用商店，请替换为正式签名并移除该行。
            signingConfig = signingConfigs.getByName("debug")
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        compose = true
    }
}

// AGP 9 内置 Kotlin：用 compilerOptions 配置 JVM 目标
kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

dependencies {
    implementation(platform("androidx.compose:compose-bom:2025.06.01"))
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.activity:activity-compose:1.9.2")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.6")
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")

    // MIUI/HyperOS 设备用 Miuix 呈现 MIUI 原生设计语言
    implementation("top.yukonga.miuix.kmp:miuix-ui-android:0.9.2")
}
