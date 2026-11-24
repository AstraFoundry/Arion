package si.astrafoundry.arion.design

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * An icon on a fixed square frame, so icons of different shapes align on the
 * same column. Brand marks keep their colors; the rest take the tint.
 */
@Composable
fun IconView(icon: ArionIcon, size: Dp = 18.dp, tint: Color = Sunbeam.colors.foreground, contentDescription: String? = null, modifier: Modifier = Modifier) {
    val painter = painterResource(icon.res)
    if (icon.original) Image(painter, contentDescription, modifier.size(size))
    else Icon(painter, contentDescription, modifier.size(size), tint = tint)
}
