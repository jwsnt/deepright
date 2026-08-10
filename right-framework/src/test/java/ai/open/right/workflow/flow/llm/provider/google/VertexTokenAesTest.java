package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.WorkflowException;
import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;

import javax.crypto.Cipher;
import java.io.InputStream;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyFactory;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.interfaces.RSAKey;
import java.security.spec.X509EncodedKeySpec;
import java.util.Arrays;
import java.util.Base64;

/**
 * 使用 {@code src/test/resources/aes} 中公钥/私钥与 {@code hello.json} / {@code hello.enc}，
 * 覆盖 {@link VertexToken#decryptCredentialBytes} 的裸 RSA 与 CMS 两条路径。
 */
public class VertexTokenAesTest {

    private static final String BASE = "/aes/";

    private static byte[] rsaEncryptPkcs1Public(PublicKey publicKey, byte[] plaintext) throws Exception {
        Cipher cipher = Cipher.getInstance("RSA/ECB/PKCS1Padding");
        cipher.init(Cipher.ENCRYPT_MODE, publicKey);
        int keyBytes = (((RSAKey) publicKey).getModulus().bitLength() + 7) / 8;
        int maxPlain = keyBytes - 11;
        java.io.ByteArrayOutputStream out = new java.io.ByteArrayOutputStream();
        for (int i = 0; i < plaintext.length; i += maxPlain) {
            int len = Math.min(maxPlain, plaintext.length - i);
            byte[] block = Arrays.copyOfRange(plaintext, i, i + len);
            out.write(cipher.doFinal(block));
        }
        return out.toByteArray();
    }

    /**
     * {@code -----BEGIN PUBLIC KEY-----}（SubjectPublicKeyInfo DER）
     */
    private static PublicKey readPublicKeySpkiPem(String classpath) throws Exception {
        String pem = new String(readAllBytes(classpath), StandardCharsets.UTF_8);
        String body = pem.trim()
                .replace("-----BEGIN PUBLIC KEY-----", "")
                .replace("-----END PUBLIC KEY-----", "")
                .replaceAll("\\s", "");
        byte[] der = Base64.getDecoder().decode(body);
        return KeyFactory.getInstance("RSA").generatePublic(new X509EncodedKeySpec(der));
    }

    @Test
    public void decrypt_invalidDerLengthAtOffset1_unaligned_throwsHelpful() throws Exception {
        PrivateKey pk = VertexTokenCrypto.parsePrivateKeyPem(new String(readAllBytes(BASE + "private.pem"), StandardCharsets.UTF_8));
        byte[] junk = new byte[50];
        junk[0] = 0x30;
        junk[1] = (byte) 0xEF;
        try {
            VertexToken.decryptCredentialBytes(pk, junk);
            Assert.fail("expected WorkflowException");
        } catch (WorkflowException e) {
            Assert.assertTrue(e.getMessage(), e.getMessage().contains("CMS decrypt failed"));
            Assert.assertTrue(e.getMessage(), e.getMessage().contains("offset 1") || e.getMessage().contains("0xef"));
        }
    }

    /**
     * 当前实现要求传入<strong>二进制</strong> CMS；Base64 需在调用方解码后再交给 {@link VertexToken#decryptCredentialBytes}。
     */
    @Test
    public void decryptCredentialBytes_binaryCms_matchesPlaintext() throws Exception {
        byte[] enc = readAllBytes(BASE + "hello.enc");
        PrivateKey pk = VertexTokenCrypto.parsePrivateKeyPem(new String(readAllBytes(BASE + "private.pem"), StandardCharsets.UTF_8));
        byte[] plain = VertexToken.decryptCredentialBytes(pk, enc);
        Assert.assertArrayEquals(readAllBytes(BASE + "hello.json"), plain);
    }

    @Test
    public void decryptCredentialBytes_afterCallerDecodesBase64_matchesPlaintext() throws Exception {
        byte[] enc = readAllBytes(BASE + "hello.enc");
        String b64 = Base64.getEncoder().encodeToString(enc);
        byte[] raw = Base64.getDecoder().decode(b64);
        PrivateKey pk = VertexTokenCrypto.parsePrivateKeyPem(new String(readAllBytes(BASE + "private.pem"), StandardCharsets.UTF_8));
        byte[] plain = VertexToken.decryptCredentialBytes(pk, raw);
        Assert.assertArrayEquals(readAllBytes(BASE + "hello.json"), plain);
    }

    @Test
    public void cmsDecrypt_helloEnc_matchesPlaintext() throws Exception {
        byte[] enc = readAllBytes(BASE + "hello.enc");
        String pem = new String(readAllBytes(BASE + "private.pem"), StandardCharsets.UTF_8);
        PrivateKey pk = VertexTokenCrypto.parsePrivateKeyPem(pem);
        byte[] plain = VertexTokenCrypto.cmsOpenSslKeyTransAesDecrypt(pk, enc);
        Assert.assertArrayEquals(readAllBytes(BASE + "hello.json"), plain);
    }

    @Test
    public void token_withKey_invokesVertexTokenPath() throws Exception {
        byte[] plain = readAllBytes(BASE + "hello.json");
        PublicKey pub = readPublicKeySpkiPem(BASE + "public.pem");
        byte[] cipher = rsaEncryptPkcs1Public(pub, plain);

        Path tmp = Files.createTempFile("vertex-token-pkcs1", ".bin");
        try {
            Files.write(tmp, cipher);
            URL cipherUrl = tmp.toUri().toURL();
            URL keyUrl = VertexTokenAesTest.class.getResource(BASE + "private.pem");
            Assert.assertNotNull(keyUrl);
            try {
                VertexToken.token(IOUtils.toByteArray(cipherUrl), IOUtils.toString(keyUrl, StandardCharsets.UTF_8), 100);
                Assert.fail("hello.json 非服务账号 JSON，预期在凭证解析或 refresh 时失败");
            } catch (Exception expected) {
                Assert.assertNotNull(expected.getMessage());
            }
        } finally {
            Files.deleteIfExists(tmp);
        }
    }

    private static byte[] readAllBytes(String classpath) throws Exception {
        try (InputStream in = VertexTokenAesTest.class.getResourceAsStream(classpath)) {
            Assert.assertNotNull("classpath 资源不存在: " + classpath, in);
            return in.readAllBytes();
        }
    }
}
