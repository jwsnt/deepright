package ai.open.right.utils;

import java.util.Base64;
import org.apache.commons.compress.compressors.gzip.GzipCompressorInputStream;
import org.apache.commons.compress.compressors.gzip.GzipCompressorOutputStream;
import org.apache.commons.io.IOUtils;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;

// 以下方法仅可用于非二进制内容
public class GzipUtils {

    // Gzip -> Base64
    public static String compressAsBase64(byte[] input) throws IOException {
        return Base64.getEncoder().encodeToString(GzipUtils.compress(input));
    }

    // Gzip -> Base64
    public static String compressAsBase64(String input) throws IOException {
        return Base64.getEncoder().encodeToString(GzipUtils.compress(input));
    }

    // Gzip
    public static byte[] compress(byte[] input) throws IOException {
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        try (GzipCompressorOutputStream gzos = new GzipCompressorOutputStream(bos)) {
            gzos.write(input);
        }
        return bos.toByteArray();
    }

    // Gzip
    public static byte[] compress(String input) throws IOException {
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        try (GzipCompressorOutputStream gzos = new GzipCompressorOutputStream(bos)) {
            gzos.write(input.getBytes(StandardCharsets.UTF_8));
        }
        return bos.toByteArray();
    }

    // Base64 -> Unzip
    public static byte[] decompressAsBase64(String input) throws IOException {
        ByteArrayInputStream bis = new ByteArrayInputStream(Base64.getDecoder().decode(input));
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        try (GzipCompressorInputStream gzis = new GzipCompressorInputStream(bis)) {
            IOUtils.copy(gzis, bos);
        }
        return bos.toByteArray();
    }

    // Unzip
    public static String decompress(byte[] input) throws IOException {
        ByteArrayInputStream bis = new ByteArrayInputStream(input);
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        try (GzipCompressorInputStream gzipInputStream = new GzipCompressorInputStream(bis)) {
            IOUtils.copy(gzipInputStream, bos);
        }
        return bos.toString(StandardCharsets.UTF_8);
    }
}