package ai.open.right.workflow.flow.llm.provider.google;

import org.springframework.util.Assert;

import javax.crypto.Cipher;
import javax.crypto.spec.IvParameterSpec;
import javax.crypto.spec.SecretKeySpec;
import java.io.ByteArrayOutputStream;
import java.math.BigInteger;
import java.nio.charset.StandardCharsets;
import java.security.KeyFactory;
import java.security.PrivateKey;
import java.security.interfaces.RSAKey;
import java.security.spec.PKCS8EncodedKeySpec;
import java.security.spec.RSAPrivateCrtKeySpec;
import java.util.Arrays;
import java.util.Base64;

/**
 * 无第三方密码库：PKCS#8 / PKCS#1 RSA 私钥 PEM、裸 RSA/PKCS#1 分块解密、
 * OpenSSL {@code cms -encrypt -aes-256-cbc -outform DER} 常见 EnvelopedData（KeyTrans + AES-CBC）解密。
 */
final class VertexTokenCrypto {

    private static final byte[] OID_PKCS7_ENVELOPED = new byte[]{0x2a, (byte) 0x86, 0x48, (byte) 0x86, (byte) 0xf7, 0x0d, 0x01, 0x07, 0x03};

    private static final byte[] OID_AES_256_CBC = new byte[]{0x60, (byte) 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x01, 0x2a};

    private VertexTokenCrypto() {
    }

    static PrivateKey parsePrivateKeyPem(String pem) throws Exception {
        String body = pem.trim()
                .replace("-----BEGIN PRIVATE KEY-----", "")
                .replace("-----END PRIVATE KEY-----", "")
                .replace("-----BEGIN RSA PRIVATE KEY-----", "")
                .replace("-----END RSA PRIVATE KEY-----", "")
                .replaceAll("\\s", "");
        byte[] der = Base64.getDecoder().decode(body);
        return pem.contains("BEGIN PRIVATE KEY") ? parsePkcs8PrivateKeyInfo(der) : parsePkcs1RsaPrivateKey(der);
    }

    private static PrivateKey parsePkcs8PrivateKeyInfo(byte[] der) throws Exception {
        Der r = new Der(der, 0, der.length);
        r.expectTag(0x30);
        int len = r.readLength();
        int end = r.pos + len;
        r.readInteger(); // version
        r.expectTag(0x30);
        int algLen = r.readLength();
        r.pos += algLen; // AlgorithmIdentifier
        r.expectTag(0x04);
        int octLen = r.readLength();
        byte[] inner = r.readBytes(octLen);
        return parsePkcs1RsaPrivateKey(inner);
    }

    private static PrivateKey parsePkcs1RsaPrivateKey(byte[] der) throws Exception {
        Der r = new Der(der, 0, der.length);
        r.expectTag(0x30);
        int len = r.readLength();
        int end = r.pos + len;
        BigInteger version = r.readInteger();
        int v = version.intValueExact();
        Assert.isTrue(v == 0 || v == 1, "Unsupported RSA private key version: " + version);
        BigInteger modulus = r.readInteger();
        BigInteger publicExponent = r.readInteger();
        BigInteger privateExponent = r.readInteger();
        BigInteger primeP = r.readInteger();
        BigInteger primeQ = r.readInteger();
        BigInteger exp1 = r.readInteger();
        BigInteger exp2 = r.readInteger();
        BigInteger crtCoefficient = r.readInteger();
        RSAPrivateCrtKeySpec spec = new RSAPrivateCrtKeySpec(
                modulus, publicExponent, privateExponent, primeP, primeQ, exp1, exp2, crtCoefficient);
        return KeyFactory.getInstance("RSA").generatePrivate(spec);
    }

    static byte[] rsaDecryptPkcs1(PrivateKey privateKey, byte[] ciphertext) throws Exception {
        Cipher cipher = Cipher.getInstance("RSA/ECB/PKCS1Padding");
        cipher.init(Cipher.DECRYPT_MODE, privateKey);
        int keyBytes = (((RSAKey) privateKey).getModulus().bitLength() + 7) / 8;
        Assert.isTrue(
                ciphertext.length % keyBytes == 0,
                "RSA ciphertext length " + ciphertext.length + " is not a multiple of key size " + keyBytes);
        try (ByteArrayOutputStream plain = new ByteArrayOutputStream()) {
            for (int offset = 0; offset < ciphertext.length; offset += keyBytes) {
                byte[] block = Arrays.copyOfRange(ciphertext, offset, offset + keyBytes);
                plain.write(cipher.doFinal(block));
            }
            return plain.toByteArray();
        }
    }

    /**
     * OpenSSL {@code cms -encrypt} DER：KeyTransRecipientInfo + EncryptedContentInfo（aes-256-cbc）。
     */
    static byte[] cmsOpenSslKeyTransAesDecrypt(PrivateKey rsaPrivate, byte[] der) throws Exception {
        Der root = new Der(der, 0, der.length);
        root.expectTag(0x30);
        int ciLen = root.readLength();
        int ciEnd = root.pos + ciLen;
        Der ci = new Der(der, root.pos, ciEnd);
        root.pos = ciEnd;

        Assert.isTrue(Arrays.equals(ci.readOid(), OID_PKCS7_ENVELOPED), "Not pkcs7-envelopedData OID");
        int t = ci.readTag();
        Assert.isTrue((t & 0xff) == 0xA0, "Expected [0] explicit EnvelopedData, tag=" + t);
        int innerLen = ci.readLength();
        int innerEnd = ci.pos + innerLen;
        Der env = new Der(der, ci.pos, innerEnd);
        ci.pos = innerEnd;

        env.expectTag(0x30);
        int envLen = env.readLength();
        int envEnd = env.pos + envLen;
        env.readInteger(); // CMSVersion

        env.expectTag(0x31); // RecipientInfos SET
        int setLen = env.readLength();
        int setEnd = env.pos + setLen;
        Der set = new Der(der, env.pos, setEnd);
        env.pos = setEnd;

        set.expectTag(0x30); // KeyTransRecipientInfo
        int recipLen = set.readLength();
        int recipEnd = set.pos + recipLen;
        Der recip = new Der(der, set.pos, recipEnd);
        set.pos = recipEnd;

        recip.readInteger(); // version
        recip.expectTag(0x30); // IssuerAndSerialNumber
        int ridLen = recip.readLength();
        recip.pos += ridLen;

        recip.expectTag(0x30); // AlgorithmIdentifier
        int algLen = recip.readLength();
        recip.pos += algLen;

        recip.expectTag(0x04);
        int encKeyLen = recip.readLength();
        byte[] encKey = recip.readBytes(encKeyLen);

        Cipher rsa = Cipher.getInstance("RSA/ECB/PKCS1Padding");
        rsa.init(Cipher.DECRYPT_MODE, rsaPrivate);
        byte[] aesKey = rsa.doFinal(encKey);

        env.expectTag(0x30); // EncryptedContentInfo
        int eciLen = env.readLength();
        int eciEnd = env.pos + eciLen;
        Der eci = new Der(der, env.pos, eciEnd);
        env.pos = eciEnd;

        eci.readOid(); // contentType data
        eci.expectTag(0x30); // contentEncryptionAlgorithm
        int ceaLen = eci.readLength();
        int ceaEnd = eci.pos + ceaLen;
        Der cea = new Der(der, eci.pos, ceaEnd);
        eci.pos = ceaEnd;

        Assert.isTrue(Arrays.equals(cea.readOid(), OID_AES_256_CBC), "Expected aes-256-cbc content encryption OID");
        cea.expectTag(0x04);
        int ivLen = cea.readLength();
        byte[] iv = cea.readBytes(ivLen);

        Assert.isTrue(eci.pos < eci.end, "Missing encryptedContent");
        int ctTag = eci.peekTag() & 0xff;
        final byte[] cipherBody;
        if ((ctTag & 0xC0) == 0x80) {
            eci.readTag();
            int ctLen = eci.readLength();
            cipherBody = eci.readBytes(ctLen);
        } else {
            Assert.isTrue(ctTag == 0x04, "Unsupported encryptedContent tag: " + ctTag);
            eci.expectTag(0x04);
            int ctLen = eci.readLength();
            cipherBody = eci.readBytes(ctLen);
        }

        Cipher aes = Cipher.getInstance("AES/CBC/PKCS5Padding");
        aes.init(Cipher.DECRYPT_MODE, new SecretKeySpec(aesKey, "AES"), new IvParameterSpec(iv));
        return aes.doFinal(cipherBody);
    }

    private static final class Der {
        final byte[] buf;
        int pos;
        final int end;

        Der(byte[] buf, int pos, int end) {
            this.buf = buf;
            this.pos = pos;
            this.end = end;
        }

        int peekTag() {
            return buf[pos] & 0xff;
        }

        void expectTag(int expected) {
            int t = readTag();
            Assert.isTrue(t == expected, "DER expected tag " + expected + " got " + t);
        }

        int readTag() {
            Assert.isTrue(pos < end, "DER truncated at tag");
            return buf[pos++] & 0xff;
        }

        int readLength() {
            Assert.isTrue(pos < end, "DER truncated at length");
            int lengthTagOffset = pos;
            int b = buf[pos++] & 0xff;
            if (b < 0x80) {
                return b;
            }
            int n = b & 0x7f;
            Assert.isTrue(
                    n >= 1 && n <= 4,
                    "DER length tag at offset " + lengthTagOffset + " is 0x" + Integer.toHexString(b)
                            + " (unsupported form: " + n + " octets). "
                            + "Usually means the input is not valid OpenSSL CMS DER, or parsing is misaligned "
                            + "(wrong key/cert, truncated file, PEM/base64 not decoded to binary, or CMS not from: "
                            + "openssl cms -encrypt -binary -aes-256-cbc -outform DER).");
            int v = 0;
            for (int i = 0; i < n; i++) {
                Assert.isTrue(pos < end, "DER truncated in long length");
                v = (v << 8) | (buf[pos++] & 0xff);
            }
            return v;
        }

        byte[] readBytes(int n) {
            Assert.isTrue(pos + n <= end, "DER read past end");
            byte[] r = Arrays.copyOfRange(buf, pos, pos + n);
            pos += n;
            return r;
        }

        BigInteger readInteger() {
            expectTag(0x02);
            int len = readLength();
            byte[] v = readBytes(len);
            return new BigInteger(1, v);
        }

        byte[] readOid() {
            expectTag(0x06);
            int len = readLength();
            return readBytes(len);
        }
    }
}
