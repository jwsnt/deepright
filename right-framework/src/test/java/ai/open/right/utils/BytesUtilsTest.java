package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

import java.nio.charset.StandardCharsets;

public class BytesUtilsTest {

    private static int jdkUtf8Length(CharSequence s) {
        return s.toString().getBytes(StandardCharsets.UTF_8).length;
    }

    @Test
    public void utf8Bytes_empty() throws Exception {
        Assert.assertEquals(0, BytesUtils.utf8Bytes(""));
    }

    @Test
    public void utf8Bytes_ascii_matchesJdk() throws Exception {
        String s = "Hello\n\t127";
        Assert.assertEquals(jdkUtf8Length(s), BytesUtils.utf8Bytes(s));
    }

    @Test
    public void utf8Bytes_twoByteBmp_matchesJdk() throws Exception {
        String s = "\u0080\u07FF";
        Assert.assertEquals(jdkUtf8Length(s), BytesUtils.utf8Bytes(s));
    }

    @Test
    public void utf8Bytes_threeByteBmp_matchesJdk() throws Exception {
        String s = "中文αβ";
        Assert.assertEquals(jdkUtf8Length(s), BytesUtils.utf8Bytes(s));
    }

    /** 增补平面（代理对），UTF-8 占 4 字节 */
    @Test
    public void utf8Bytes_supplementary_matchesJdk() throws Exception {
        String s = "\uD83D\uDE00";
        Assert.assertEquals(4, BytesUtils.utf8Bytes(s));
        Assert.assertEquals(jdkUtf8Length(s), BytesUtils.utf8Bytes(s));
    }

    @Test
    public void utf8Bytes_mixed_matchesJdk() throws Exception {
        String s = "a\u0080\u0800\uD83D\uDE00z";
        Assert.assertEquals(jdkUtf8Length(s), BytesUtils.utf8Bytes(s));
    }

    @Test
    public void utf8Bytes_stringBuilderCharSequence_matchesJdk() throws Exception {
        StringBuilder sb = new StringBuilder("x").append('\u4e2d').append("\uD83D\uDE00");
        Assert.assertEquals(jdkUtf8Length(sb), BytesUtils.utf8Bytes(sb));
    }
}
