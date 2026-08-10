package ai.open.right.workflow.flow.media;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

public class MediaTransferUtilsTest {

    @Test
    public void uriConstantContainsExpectedSchemes() {
        assertEquals(4, MediaTransferUtils.URI.size());
        assertTrue(MediaTransferUtils.URI.contains("https"));
        assertTrue(MediaTransferUtils.URI.contains("http"));
        assertTrue(MediaTransferUtils.URI.contains("ftp"));
        assertTrue(MediaTransferUtils.URI.contains("ftps"));
    }

    @Test
    public void testIsNetworkWithHttps() throws Exception {
        String uri = "https://www.example.com";
        assertTrue(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithHttp() throws Exception {
        String uri = "http://localhost:8080";
        assertTrue(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithFtp() throws Exception {
        String uri = "ftp://192.168.1.1/resource";
        assertTrue(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithFtps() throws Exception {
        String uri = "ftps://secure-storage.io";
        assertTrue(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithUpperCase() throws Exception {
        String uri = "HTTPS://WWW.EXAMPLE.COM";
        assertTrue(MediaTransferUtils.isNetwork(uri));
    }

    /** 合法 URI、scheme 不在白名单 → false */
    @Test
    public void testIsNetworkWithFileScheme() throws Exception {
        String uri = "file:///etc/hosts";
        assertFalse(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithMailtoScheme() throws Exception {
        assertFalse(MediaTransferUtils.isNetwork("mailto:user@example.com"));
    }

    /** 合法 URI、无 scheme（相对路径）→ getScheme() 为 null → false */
    @Test
    public void testIsNetworkWithNoScheme() throws Exception {
        String uri = "/home/user/data.mp4";
        assertFalse(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithRelativePathOnly() throws Exception {
        assertFalse(MediaTransferUtils.isNetwork("path/only/no-scheme"));
    }

    /** 空串解析失败 → URISyntaxException → false */
    @Test
    public void testIsNetworkWithEmptyString() throws Exception {
        String uri = "";
        assertFalse(MediaTransferUtils.isNetwork(uri));
    }

    /** 非法 URI → URISyntaxException → false（不向外抛出） */
    @Test
    public void testIsNetworkWithInvalidUriReturnsFalse() throws Exception {
        String uri = "https://example.com/a b c";
        assertFalse(MediaTransferUtils.isNetwork(uri));
    }

    @Test
    public void testIsNetworkWithMalformedUriReturnsFalse() throws Exception {
        assertFalse(MediaTransferUtils.isNetwork("http://["));
    }

    /** null 导致 URI 构造阶段 NPE → 落入 catch(Exception) → false */
    @Test
    public void testIsNetworkWithNullReturnsFalse() throws Exception {
        assertFalse(MediaTransferUtils.isNetwork(null));
    }
}
