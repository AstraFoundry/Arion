package si.astrafoundry.arion.ui

import si.astrafoundry.arion.design.ArionIcon
import si.astrafoundry.arion.design.IconView

import android.content.Context
import android.content.Intent
import android.graphics.BitmapFactory
import android.net.Uri
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.painter.Painter
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.text.LinkAnnotation
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.withLink
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import si.astrafoundry.arion.core.AccountConfig
import si.astrafoundry.arion.core.AccountFailure
import si.astrafoundry.arion.core.AccountPhase
import si.astrafoundry.arion.core.AccountProfile
import si.astrafoundry.arion.core.AccountServing
import si.astrafoundry.arion.core.AccountState
import si.astrafoundry.arion.core.UsageSharing
import si.astrafoundry.arion.design.R
import si.astrafoundry.arion.design.Sunbeam
import java.net.HttpURLConnection
import java.net.URL
import java.util.Locale
import si.astrafoundry.arion.R as AppR

/** Arion's site pages the sign-in links to, localized as the desktop's. */
object SiteLinks {
    private const val SITE = "https://arion.astrafoundry.si"
    private val origin get() = if (Locale.getDefault().language == "zh") "$SITE/zh" else SITE
    val terms get() = "$origin/terms/"
    val privacy get() = "$origin/privacy/"
    val usageData get() = "$origin/usage-data/"
}

fun open(context: Context, url: String) {
    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
}

/** The account's sign-in methods. Every method opens the same browser sign-in. */
@Composable
fun SignInMethods(enabled: Boolean, signIn: () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlineButton(stringResource(R.string.account_continue_google), ArionIcon.Google, enabled = enabled, onClick = signIn)
        OutlineButton(stringResource(R.string.account_continue_git_hub), ArionIcon.Github, enabled = enabled, onClick = signIn)
        OutlineButton(stringResource(R.string.account_continue_email), ArionIcon.Mail, enabled = enabled, onClick = signIn)
    }
}

/** A bordered button on the page: the card's face with a hairline edge. */
@Composable
fun OutlineButton(label: String, icon: ArionIcon?, enabled: Boolean = true, onClick: () -> Unit) {
    val c = Sunbeam.colors
    val shape = RoundedCornerShape(14.dp)
    Row(
        Modifier.fillMaxWidth().height(48.dp).alpha(if (enabled) 1f else 0.45f).clip(shape).background(c.card).border(Dp.Hairline, c.border, shape)
            .clickable(enabled = enabled, role = Role.Button, onClick = onClick),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) {
            IconView(icon, 18.dp)
            Spacer(Modifier.width(10.dp))
        }
        Text(label, style = Sunbeam.type.label, color = c.foreground)
    }
}

/**
 * A checkbox with a label whose named words link to documents. The links
 * keep their own taps; the rest of the row toggles the box.
 */
@Composable
fun ConsentRow(label: String, links: List<Pair<String, String>>, checked: Boolean, onChange: (Boolean) -> Unit) {
    val c = Sunbeam.colors
    val text = buildAnnotatedString {
        var rest = label
        for ((word, url) in links) {
            val at = rest.indexOf(word)
            check(at >= 0) { "consent link is missing from its localized label" }
            append(rest.substring(0, at))
            withLink(LinkAnnotation.Url(url, TextLinkStyles(SpanStyle(textDecoration = TextDecoration.Underline)))) { append(word) }
            rest = rest.substring(at + word.length)
        }
        append(rest)
    }
    Row(
        Modifier.fillMaxWidth().clickable(role = Role.Checkbox) { onChange(!checked) }
            .semantics { stateDescription = if (checked) "1" else "0" }.padding(vertical = 4.dp),
        verticalAlignment = Alignment.Top,
    ) {
        IconView(if (checked) ArionIcon.SquareCheck else ArionIcon.Square, 20.dp)
        Spacer(Modifier.width(10.dp))
        Text(text, style = Sunbeam.type.caption, color = c.foreground, modifier = Modifier.padding(top = 1.dp))
    }
}

/** The profile picture, or the initials, or the person glyph while signed out. */
@Composable
fun Avatar(profile: AccountProfile?, size: Dp) {
    val c = Sunbeam.colors
    var picture by remember(profile?.picture) { mutableStateOf<ImageBitmap?>(null) }
    LaunchedEffect(profile?.picture) {
        val url = profile?.picture?.takeIf { it.startsWith("https://") } ?: return@LaunchedEffect
        // At most 2 MiB, as the desktop; a failure leaves the initials.
        picture = withContext(Dispatchers.IO) {
            runCatching {
                val connection = URL(url).openConnection() as HttpURLConnection
                connection.connectTimeout = 15000
                connection.readTimeout = 15000
                connection.inputStream.use { input ->
                    val bytes = input.readNBytes((2 shl 20) + 1)
                    if (connection.responseCode != 200 || bytes.size > (2 shl 20)) null
                    else BitmapFactory.decodeByteArray(bytes, 0, bytes.size)?.asImageBitmap()
                }
            }.getOrNull()
        }
    }
    Box(Modifier.size(size).clip(CircleShape).background(c.secondary), contentAlignment = Alignment.Center) {
        when {
            picture != null -> Image(picture!!, null, Modifier.size(size), contentScale = ContentScale.Crop)
            profile != null -> Text(profile.initials, color = c.secondaryForeground, fontWeight = FontWeight.SemiBold, fontSize = (size.value * 0.38f).sp)
            else -> IconView(ArionIcon.User, size * 0.5f, c.mutedForeground)
        }
    }
}

/** The account row at the top of Settings. */
@Composable
fun AccountRow(state: AccountState, onClick: () -> Unit) {
    val c = Sunbeam.colors
    val profile = state.profile.takeIf { state.phase != AccountPhase.SignedOut && state.phase != AccountPhase.SigningIn }
    val name = profile?.let { it.name.ifEmpty { it.email } } ?: stringResource(R.string.account_sign_in_title)
    val detail = when {
        profile == null -> null
        state.phase == AccountPhase.Restoring -> stringResource(R.string.account_restoring)
        profile.name.isEmpty() -> stringResource(R.string.account_signed_in)
        else -> profile.email
    }
    Section {
        Row(
            Modifier.fillMaxWidth().clickable(role = Role.Button, onClick = onClick).padding(horizontal = 16.dp, vertical = 10.dp)
                .semantics(mergeDescendants = true) { contentDescription = listOfNotNull(name, detail).joinToString(", ") },
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Avatar(profile, 44.dp)
            Spacer(Modifier.width(14.dp))
            Column(Modifier.weight(1f)) {
                Text(name, style = Sunbeam.type.body.copy(fontWeight = FontWeight.SemiBold), color = c.foreground)
                if (detail != null) Text(detail, style = Sunbeam.type.caption, color = c.mutedForeground)
            }
            IconView(ArionIcon.ChevronRight, 16.dp, c.faint)
        }
    }
}

/** The account: sign in, or the profile with usage sharing, the portal and Sign out. */
@Composable
fun AccountScreen(account: AccountServing, state: AccountState) {
    val c = Sunbeam.colors
    val context = LocalContext.current
    val failure = state.failure?.let { stringResource(failureText(it)) }
    when (state.phase) {
        AccountPhase.SignedOut, AccountPhase.SigningIn -> Column(Modifier.padding(horizontal = 32.dp, vertical = 8.dp)) {
            Text(stringResource(R.string.account_sign_in_title), style = Sunbeam.type.title, color = c.foreground)
            Spacer(Modifier.height(24.dp))
            if (state.phase == AccountPhase.SigningIn) {
                Text(stringResource(R.string.account_waiting), style = Sunbeam.type.body, color = c.secondaryForeground)
                Spacer(Modifier.height(16.dp))
                OutlineButton(stringResource(R.string.account_cancel), null) { account.cancel() }
            } else {
                SignInMethods(enabled = true) { account.signIn(shareUsage = false) }
                Spacer(Modifier.height(16.dp))
                LinkedText(stringResource(R.string.account_terms_notice), listOf(
                    stringResource(R.string.account_terms) to SiteLinks.terms,
                    stringResource(R.string.account_privacy) to SiteLinks.privacy,
                ))
            }
            if (failure != null) {
                Spacer(Modifier.height(16.dp))
                Text(failure, style = Sunbeam.type.caption, color = c.danger)
            }
        }
        else -> Column {
            Column(Modifier.fillMaxWidth().padding(vertical = 12.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                Avatar(state.profile, 72.dp)
                Spacer(Modifier.height(14.dp))
                Text(state.profile?.let { it.name.ifEmpty { it.email } }.orEmpty(), style = Sunbeam.type.title, color = c.foreground, textAlign = TextAlign.Center)
                val detail = when {
                    state.phase == AccountPhase.Restoring -> stringResource(R.string.account_restoring)
                    state.profile?.let { it.name.isNotEmpty() && it.email.isNotEmpty() } == true -> state.profile!!.email
                    else -> stringResource(R.string.account_signed_in)
                }
                Text(detail, style = Sunbeam.type.bodySmall, color = c.mutedForeground, textAlign = TextAlign.Center)
            }
            if (state.signedIn) {
                Section {
                    SettingRow(stringResource(R.string.account_usage_share), trailing = {
                        Toggle(state.usageSharing == UsageSharing.Shared, enabled = !state.busy && state.usageSharing != UsageSharing.Loading) {
                            account.setUsageSharing(it)
                        }
                    })
                }
                val portal = AccountConfig().portal
                Section(stringResource(R.string.account_portal)) {
                    listOf(
                        R.string.account_portal_profile to "profile",
                        R.string.account_portal_security to "security",
                        R.string.account_portal_connections to "connections",
                        R.string.account_portal_products to "products",
                        R.string.account_portal_data to "data",
                    ).forEachIndexed { index, (label, path) ->
                        SettingRow(stringResource(label), divider = index > 0, onClick = { open(context, "$portal/$path") }, trailing = {
                            IconView(ArionIcon.ExternalLink, 16.dp, c.faint)
                        })
                    }
                }
            }
            Section(footer = failure) {
                Row(
                    Modifier.fillMaxWidth().height(52.dp).clickable(enabled = !state.busy || state.phase == AccountPhase.Restoring, role = Role.Button) { account.signOut() },
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(stringResource(R.string.account_sign_out), style = Sunbeam.type.body, color = if (state.busy && state.phase != AccountPhase.Restoring) c.faint else c.danger)
                }
            }
        }
    }
}

/** Muted caption text whose named words open documents. */
@Composable
fun LinkedText(label: String, links: List<Pair<String, String>>) {
    val c = Sunbeam.colors
    val text = buildAnnotatedString {
        append(label)
        for ((word, url) in links) {
            val at = label.indexOf(word)
            if (at >= 0) addLink(LinkAnnotation.Url(url, TextLinkStyles(SpanStyle(color = c.foreground, textDecoration = TextDecoration.Underline))), at, at + word.length)
        }
    }
    Text(text, style = Sunbeam.type.caption, color = c.mutedForeground)
}

fun failureText(failure: AccountFailure): Int = when (failure) {
    AccountFailure.InProgress -> R.string.account_error_in_progress
    AccountFailure.Denied -> R.string.account_error_denied
    AccountFailure.Authorization -> R.string.account_error_authorization
    AccountFailure.Token -> R.string.account_error_token
    AccountFailure.Discovery -> R.string.account_error_discovery
    AccountFailure.Store -> R.string.account_error_store
    AccountFailure.SessionEnded -> R.string.account_error_session_ended
    AccountFailure.Refresh -> R.string.account_error_refresh
    AccountFailure.RevokeUnconfirmed -> R.string.account_error_revoke_unconfirmed
    AccountFailure.UsageConsent -> R.string.account_error_usage_consent
    AccountFailure.Generic -> R.string.account_error_generic
}
