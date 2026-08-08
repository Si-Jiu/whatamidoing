package dev.sijiu49.waid

import android.app.AppOpsManager
import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.provider.Settings
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import dev.sijiu49.waid.ui.AppTheme

class MainActivity : ComponentActivity() {
    private val cfg by lazy { ConfigStore(this) }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            AppTheme {
                if (isMiui()) {
                    MiuixSettingsScreen(
                        cfg = cfg,
                        onEnabledToggle = { enabled -> onSharingToggled(enabled) }
                    )
                } else {
                    SettingsScreen(
                        cfg = cfg,
                        onEnabledToggle = { enabled -> onSharingToggled(enabled) }
                    )
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        // 用户从「使用情况访问」设置页返回后，若已授权则自动开始共享。
        if (cfg.enabled && hasUsageAccess()) {
            startSharing()
        }
    }

    private fun onSharingToggled(enabled: Boolean) {
        cfg.enabled = enabled
        if (enabled) {
            if (hasUsageAccess()) {
                startSharing()
            } else {
                promptUsageAccess()
            }
        } else {
            stopSharing()
        }
    }

    private fun startSharing() {
        startForegroundService(Intent(this, ShareService::class.java))
    }

    private fun stopSharing() {
        stopService(Intent(this, ShareService::class.java))
    }

    private fun promptUsageAccess() {
        Toast.makeText(this, "请允许「使用情况访问」，之后将自动开始共享", Toast.LENGTH_LONG).show()
        startActivity(Intent(Settings.ACTION_USAGE_ACCESS_SETTINGS))
    }

    private fun hasUsageAccess(): Boolean {
        val appOps = getSystemService(Context.APP_OPS_SERVICE) as AppOpsManager
        val mode = appOps.checkOpNoThrow(
            AppOpsManager.OPSTR_GET_USAGE_STATS,
            android.os.Process.myUid(),
            packageName
        )
        return mode == AppOpsManager.MODE_ALLOWED
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(cfg: ConfigStore, onEnabledToggle: (Boolean) -> Unit) {
    var serverUrl by rememberSaveable { mutableStateOf(cfg.serverUrl) }
    var token by rememberSaveable { mutableStateOf(cfg.token) }
    var interval by rememberSaveable { mutableStateOf(cfg.intervalSecs.toString()) }
    var enabled by rememberSaveable { mutableStateOf(cfg.enabled) }
    var saved by remember { mutableStateOf(false) }

    Scaffold(
        topBar = { TopAppBar(title = { Text("whatamidoing") }) }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(20.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Card {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text("共享前台状态", style = MaterialTheme.typography.titleMedium)
                    Switch(
                        checked = enabled,
                        onCheckedChange = {
                            enabled = it
                            onEnabledToggle(it)
                        }
                    )
                }
            }

            OutlinedTextField(
                value = serverUrl,
                onValueChange = { serverUrl = it },
                label = { Text("服务端地址") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth()
            )
            OutlinedTextField(
                value = token,
                onValueChange = { token = it },
                label = { Text("设备 Token（管理面板添加设备后复制）") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth()
            )
            OutlinedTextField(
                value = interval,
                onValueChange = { interval = it },
                label = { Text("上报间隔（秒）") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth()
            )

            Button(
                onClick = {
                    cfg.serverUrl = serverUrl.trim()
                    cfg.token = token.trim()
                    cfg.intervalSecs = interval.toIntOrNull() ?: 5
                    saved = true
                    onEnabledToggle(cfg.enabled)
                },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(if (saved) "已保存 ✓" else "保存")
            }
        }
    }
}
