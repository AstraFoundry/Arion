package si.astrafoundry.arion.storage

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.KeyStore
import java.util.Base64
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * Secrets (API keys, the refresh token) encrypted with an AES-GCM key that
 * never leaves the Android Keystore, stored in private preferences that are
 * excluded from backups.
 */
class Secrets(context: Context) {
    private val store = context.getSharedPreferences("arion.secrets", Context.MODE_PRIVATE)

    private fun key(): SecretKey {
        val keystore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        (keystore.getEntry(ALIAS, null) as? KeyStore.SecretKeyEntry)?.let { return it.secretKey }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore")
        generator.init(
            KeyGenParameterSpec.Builder(ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .build(),
        )
        return generator.generateKey()
    }

    fun read(name: String): String? {
        val stored = store.getString(name, null) ?: return null
        val bytes = Base64.getDecoder().decode(stored)
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, bytes, 0, IV))
        return String(cipher.doFinal(bytes, IV, bytes.size - IV), Charsets.UTF_8)
    }

    fun write(name: String, value: String) {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, key())
        val sealed = cipher.iv + cipher.doFinal(value.toByteArray(Charsets.UTF_8))
        if (!store.edit().putString(name, Base64.getEncoder().encodeToString(sealed)).commit()) error("could not save $name")
    }

    fun delete(name: String) {
        if (!store.edit().remove(name).commit()) error("could not delete $name")
    }

    fun has(name: String): Boolean = store.contains(name)

    private companion object {
        const val ALIAS = "arion.secrets"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val IV = 12
    }
}
