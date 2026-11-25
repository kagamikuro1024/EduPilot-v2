package vn.edupilot.app

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.location.LocationManager
import android.os.Build
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.security.keystore.StrongBoxUnavailableException
import android.util.Base64
import com.getcapacitor.JSArray
import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.PrivateKey
import java.security.Signature
import java.security.spec.ECGenParameterSpec

/**
 * Khoá thiết bị cho điểm danh bằng mã (US-AS-11 AC4, ARCHITECTURE §11.7). Khoá EC P-256 `edupilot.checkin.v1` nằm trong Android Keystore, không xuất được.
 * Trang `/checkin` gọi `window.Capacitor.nativePromise("EduPilotAttest", …)`:
 *  - registerDevice({nonce}) → {public_key (SPKI base64), attestation_chain[] (X.509 DER base64, lá trước)}: tạo LẠI khoá với challenge = nonce;
 *  - sign({message}) → {signature (DER base64), public_key}: ký UTF-8 `message` bằng SHA256withECDSA; chưa có khoá thì tạo (không challenge);
 *  - isMockLocation() → {mock}: `Location.isMock()` của vị trí gần nhất đã biết.
 * Không log message, chữ ký hay toạ độ.
 */
@CapacitorPlugin(name = "EduPilotAttest")
class EduPilotAttestPlugin : Plugin() {
    private val alias = "edupilot.checkin.v1"

    private fun store(): KeyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }

    private fun generate(challenge: ByteArray?) {
        fun spec(strongBox: Boolean): KeyGenParameterSpec {
            val b = KeyGenParameterSpec.Builder(alias, KeyProperties.PURPOSE_SIGN)
                .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
                .setDigests(KeyProperties.DIGEST_SHA256)
            if (challenge != null) b.setAttestationChallenge(challenge)
            if (strongBox && Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) b.setIsStrongBoxBacked(true)
            return b.build()
        }
        val gen = KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_EC, "AndroidKeyStore")
        try {
            gen.initialize(spec(true)) // StrongBox trước, lùi về TEE
            gen.generateKeyPair()
        } catch (e: StrongBoxUnavailableException) {
            gen.initialize(spec(false))
            gen.generateKeyPair()
        }
    }

    private fun b64(b: ByteArray) = Base64.encodeToString(b, Base64.NO_WRAP)

    @PluginMethod
    fun registerDevice(call: PluginCall) {
        val nonce = call.getString("nonce") ?: return call.reject("thiếu nonce")
        try {
            val challenge = Base64.decode(nonce, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
            val ks = store()
            if (ks.containsAlias(alias)) ks.deleteEntry(alias)
            generate(challenge)
            val chain = ks.getCertificateChain(alias) ?: return call.reject("không có chuỗi chứng chỉ")
            val out = JSObject()
            out.put("public_key", b64(chain[0].publicKey.encoded))
            val arr = JSArray()
            chain.forEach { arr.put(b64(it.encoded)) }
            out.put("attestation_chain", arr)
            call.resolve(out)
        } catch (e: Exception) {
            call.reject("registerDevice lỗi")
        }
    }

    @PluginMethod
    fun sign(call: PluginCall) {
        val message = call.getString("message") ?: return call.reject("thiếu message")
        try {
            val ks = store()
            if (!ks.containsAlias(alias)) generate(null)
            val key = ks.getKey(alias, null) as PrivateKey
            val sig = Signature.getInstance("SHA256withECDSA").run {
                initSign(key)
                update(message.toByteArray(Charsets.UTF_8))
                sign() // DER
            }
            val out = JSObject()
            out.put("signature", b64(sig))
            out.put("public_key", b64(ks.getCertificate(alias).publicKey.encoded))
            call.resolve(out)
        } catch (e: Exception) {
            call.reject("sign lỗi")
        }
    }

    @PluginMethod
    fun isMockLocation(call: PluginCall) {
        val out = JSObject()
        out.put("mock", mockNow(context))
        call.resolve(out)
    }

    private fun mockNow(ctx: Context): Boolean {
        // «Xấp xỉ» chỉ cấp COARSE: vẫn đọc được vị trí cuối của nhà cung cấp mạng; nhà cung cấp GPS ném SecurityException và bị runCatching bỏ qua.
        val granted = listOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION).any { ctx.checkSelfPermission(it) == PackageManager.PERMISSION_GRANTED }
        if (!granted) return false
        val lm = ctx.getSystemService(Context.LOCATION_SERVICE) as LocationManager
        val best: Location? = lm.getProviders(true).mapNotNull { runCatching { lm.getLastKnownLocation(it) }.getOrNull() }.maxByOrNull { it.time }
        best ?: return false
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) best.isMock else @Suppress("DEPRECATION") best.isFromMockProvider
    }
}
