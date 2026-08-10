package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

import java.io.IOException;

public class GzipUtilsTest {

    @Test
    public void testDecompress() throws Exception {
        String source = "Hello World";
        String target = GzipUtils.decompress(GzipUtils.compress(source));
        Assert.assertEquals(source, target);
    }

    @Test
    public void testDecompressAsBase64WithString() throws Exception {
        String source = "Hello World";
        byte[] target = GzipUtils.decompressAsBase64(GzipUtils.compressAsBase64(source));
        Assert.assertEquals(source, new String(target));
    }

    @Test
    public void testDecompressAsBase64WithBytes() throws Exception {
        String source = "Hello World";
        byte[] target = GzipUtils.decompressAsBase64(GzipUtils.compressAsBase64(source.getBytes()));
        Assert.assertEquals(source, new String(target));
    }

    @Test
    public void testCompressAndDecompress() throws IOException {
        Assert.assertEquals(GzipUtils.decompress(GzipUtils.compress("Hello World")), "Hello World");
    }

    @Test
    public void testEmptyInput() throws IOException {
        // Test compress and decompress with an empty string
        String source = "";
        String target = GzipUtils.decompress(GzipUtils.compress(source));
        Assert.assertEquals(source, target);
    }

    @Test(expected = IOException.class)
    public void testDecompressInvalidData() throws IOException {
        // Test decompress with invalid gzip data, should throw IOException
        byte[] invalidData = "invalid gzip data".getBytes();
        GzipUtils.decompress(invalidData);
    }

    @Test
    public void testLargeInput() throws IOException {
        // Test compress and decompress with a large string
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < 10000; i++) {
            sb.append("Hello World ");
        }
        String source = sb.toString();
        String target = GzipUtils.decompress(GzipUtils.compress(source));
        Assert.assertEquals(source, target);
    }

    @org.junit.jupiter.api.Test
    public void testCompressEmpty() throws java.io.IOException {
        byte[] compressed = GzipUtils.compress(new byte[0]);
        org.junit.jupiter.api.Assertions.assertNotNull(compressed);
    }

    @org.junit.jupiter.api.Test
    public void testUncompressInvalidBase64() {
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            GzipUtils.decompressAsBase64("invalid-base64-string!!!");
        });
    }

    @org.junit.jupiter.api.Test
    public void testCompressNullInput() {
        // 边界测试：压缩 null 字节数组
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            GzipUtils.compress((byte[]) null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testDecompressNullInput() {
        // 边界测试：解压 null 字节数组
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            GzipUtils.decompress(null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testCompressAsBase64NullInput() {
        // 边界测试：Base64压缩 null 输入
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            GzipUtils.compressAsBase64((byte[]) null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testDecompressAsBase64NullInput() {
        // 边界测试：Base64解压 null 字符串
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            GzipUtils.decompressAsBase64((String) null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testSpecialCharacters() throws java.io.IOException {
        // 边界测试：包含特殊字符和多字节字符的字符串
        String source = "!@#$%^&*()_+ \n\t 中文测试 🚀";
        String target = GzipUtils.decompress(GzipUtils.compress(source));
        org.junit.jupiter.api.Assertions.assertEquals(source, target);
    }

    @org.junit.jupiter.api.Test
    public void testCompressStringNull() {
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            GzipUtils.compress((String) null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testDecompressEmptyBytes() throws java.io.IOException {
        // 预期抛出 EOFException 或 IOException，因为空字节数组不是合法的 Gzip 流
        org.junit.jupiter.api.Assertions.assertThrows(java.io.IOException.class, () -> {
            GzipUtils.decompress(new byte[0]);
        });
    }

}

