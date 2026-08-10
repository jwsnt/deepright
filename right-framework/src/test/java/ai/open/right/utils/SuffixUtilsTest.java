package ai.open.right.utils;

import org.apache.commons.lang3.StringUtils;
import org.junit.Assert;
import org.junit.Test;

import java.nio.file.Files;
import java.nio.file.Paths;

/**
 * SuffixUtils 单元测试。
 * isBinary 依赖 Files.probeContentType，常见扩展名在标准 JDK 下可预期。
 */
public class SuffixUtilsTest {

    @Test
    public void testIsBinaryWhenSuffixIsNull() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary(null));
    }

    @Test
    public void testIsBinaryWhenSuffixIsEmpty() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary(""));
    }

    @Test
    public void testIsBinaryWhenSuffixIsJpeg() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary("jpeg"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsJpg() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary("jpg"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsPng() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary("png"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsZip() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary("zip"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsPdf() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary("pdf"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsPpt() throws Exception {
        Assert.assertTrue(SuffixUtils.isBinary("ppt"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsTxt() throws Exception {
        Assert.assertFalse(SuffixUtils.isBinary("txt"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsXml() throws Exception {
        Assert.assertFalse(SuffixUtils.isBinary("xml"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsJson() throws Exception {
        Assert.assertFalse(SuffixUtils.isBinary("json"));
    }

    @Test
    public void testIsBinaryWhenSuffixIsUnknownExtension() throws Exception {
        Assert.assertFalse(SuffixUtils.isBinary("unknown_ext"));
    }

    /**
     * 白名单内且当前 JDK 对 {@code t.扩展名} 能探测出非空类型的扩展名，应与 {@code isBinary} 一致为 true。
     * 若某后缀在目标平台上探测为 null，则白名单里虽有占位（含 null）也不会判为二进制。
     */
    @Test
    public void testIsBinaryForAdditionalProbedWhitelistedBinarySuffixes() throws Exception {
        String[] moreBinary = {
            "mp3", "mp4", "mov", "avi", "wav", "flac",
            "xlsx", "docx", "pptx", "xls", "doc",
            "rar", "7z", "tar", "gz", "bz2", "db",
            "class", "pyc", "exe", "dll", "bin", "iso", "dat", "o", "so", "elf", "obj", "ttf", "otf", "bmp", "psd", "swf", "torrent", "sqlite"
        };
        for (String s : moreBinary) {
            String ct = Files.probeContentType(Paths.get("t." + s));
            if (StringUtils.isNotEmpty(ct) && SuffixUtils.MIME_TYPE.contains(ct)) {
                Assert.assertTrue("expected binary: " + s, SuffixUtils.isBinary(s));
            }
        }
    }

    /** 常见纯文本/结构化文本扩展名：探测类型不在白名单或探测为空，应为非二进制。 */
    @Test
    public void testIsBinaryFalseForTextLikeSuffixes() throws Exception {
        String[] notBinary = {
            "md", "yml", "yaml", "html", "css", "java", "py", "properties", "csv", "log", "sh"
        };
        for (String s : notBinary) {
            String ct = Files.probeContentType(Paths.get("t." + s));
            if (StringUtils.isNotEmpty(ct) && SuffixUtils.MIME_TYPE.contains(ct)) {
                Assert.assertTrue("probed MIME is whitelisted, expect binary: " + s, SuffixUtils.isBinary(s));
            } else {
                Assert.assertFalse("expected not binary: " + s, SuffixUtils.isBinary(s));
            }
        }
    }

    /** 仅空白字符时非 isEmpty，按路径探测；通常非白名单，应为 false。 */
    @Test
    public void testIsBinaryWhenSuffixIsWhitespace() throws Exception {
        Assert.assertFalse(SuffixUtils.isBinary("   "));
    }

    /** 扩展名应不含前导点号，否则 path 为 {@code t..ext}，与 “无点扩展名” 的约定不一致。 */
    @Test
    public void testIsBinaryWhenSuffixHasLeadingDot() throws Exception {
        Assert.assertFalse(SuffixUtils.isBinary(".txt"));
    }

    @Test
    public void testMimeTypeSetIsInitialized() {
        Assert.assertNotNull(SuffixUtils.MIME_TYPE);
    }
}
