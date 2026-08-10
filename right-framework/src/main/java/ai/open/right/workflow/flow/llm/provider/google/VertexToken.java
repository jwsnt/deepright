/// /////////////////////////////////////////////////////////////
/// <dependency>
///     <groupId>com.google.api-client</groupId>
///     <artifactId>google-api-client</artifactId>
///     <version>${google.version}</version>
///     <scope>compile</scope>
/// </dependency>
/// <dependency>
///     <groupId>com.google.oauth-client</groupId>
///     <artifactId>google-oauth-client</artifactId>
///     <version>${google.version}</version>
///     <scope>compile</scope>
/// </dependency>
/// /////////////////////////////////////////////////////////////
/// openssl genrsa -out private.pem 2048
/// openssl rsa -in private.pem -pubout -out public.pem
/// openssl req -x509 -new -key private.pem -out cert.pem -days 365 -subj "/C=CN/ST=Default/L=Default/O=Default/CN=ServiceAccount"
/// openssl cms -encrypt -binary -aes-256-cbc -in service-account.json -out service-account.enc -outform DER cert.pem
/// /////////////////////////////////////////////////////////////
package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.WorkflowException;
import com.google.api.client.googleapis.auth.oauth2.GoogleCredential;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.security.PrivateKey;
import java.security.interfaces.RSAKey;
import java.util.Arrays;

@Slf4j
public class VertexToken {

    public static final String SCOPE_1 = "https://www.googleapis.com/auth/generative-language.retriever";

    public static final String SCOPE_2 = "https://www.googleapis.com/auth/cloud-platform";

    public static String token(byte[] credit, String key, Integer seconds) throws Exception {
        if (key != null) {
            PrivateKey privateKey = VertexTokenCrypto.parsePrivateKeyPem(key);
            byte[] plain = VertexToken.decryptCredentialBytes(privateKey, credit);
            try (InputStream creditStream = new ByteArrayInputStream(plain)) {
                return VertexToken.credentialFromStream(IOUtils.toString(creditStream, StandardCharsets.UTF_8), seconds);
            }
        }
        return VertexToken.credentialFromStream(new String(credit, StandardCharsets.UTF_8), seconds);
    }

    protected static String credentialFromStream(String credit, Integer seconds) throws Exception {
        GoogleCredential credential = GoogleCredential.fromStream(new ByteArrayInputStream(credit.getBytes(StandardCharsets.UTF_8))).createScoped(Arrays.asList(VertexToken.SCOPE_1, VertexToken.SCOPE_2));
        credential.refreshToken();
        credential.setExpiresInSeconds(seconds.longValue());
        String token = credential.getAccessToken();
        Assert.hasText(token, "Vertex token can not be empty");
        if (log.isDebugEnabled()) {
            log.debug("Vertex token={}", StringUtils.repeat("*", StringUtils.length(token)));
        }
        return token;
    }

    protected static byte[] decryptCredentialBytes(PrivateKey privateKey, byte[] raw) throws Exception {
        int keyBytes = (((RSAKey) privateKey).getModulus().bitLength() + 7) / 8;
        boolean derSequence = raw.length >= 2 && (raw[0] & 0xFF) == 0x30;
        if (derSequence) {
            try {
                return VertexTokenCrypto.cmsOpenSslKeyTransAesDecrypt(privateKey, raw);
            } catch (Exception cmsFail) {
                if (raw.length % keyBytes == 0) {
                    return VertexTokenCrypto.rsaDecryptPkcs1(privateKey, raw);
                }
                throw new WorkflowException("CMS decrypt failed: " + cmsFail.getMessage());
            }
        }
        if (raw.length % keyBytes != 0) {
            return VertexTokenCrypto.cmsOpenSslKeyTransAesDecrypt(privateKey, raw);
        } else {
            return VertexTokenCrypto.rsaDecryptPkcs1(privateKey, raw);
        }
    }
}
