package ai.deepright.utils;

import java.nio.CharBuffer;

public class SecurityTruncater {

    // CharBuffer.wrap(content)零拷贝遍历，并按字符累计UTF-8字节数后返回安全子串
    public static String truncate(String content, Integer truncate) throws Exception {
        CharBuffer chars = CharBuffer.wrap(content);
        int maxBytes = truncate;
        int usedBytes = 0;
        int endIndex = 0;
        while (endIndex < chars.length()) {
            char current = chars.get(endIndex);
            int charCount = 1;
            int utf8Bytes;
            if (Character.isHighSurrogate(current)) {
                if (endIndex + 1 >= chars.length()) {
                    break;
                }
                char next = chars.get(endIndex + 1);
                if (!Character.isLowSurrogate(next)) {
                    break;
                }
                utf8Bytes = utf8Bytes(Character.toCodePoint(current, next));
                charCount = 2;
            } else if (Character.isLowSurrogate(current)) {
                break;
            } else {
                utf8Bytes = utf8Bytes(current);
            }
            if (usedBytes + utf8Bytes > maxBytes) {
                break;
            }
            usedBytes += utf8Bytes;
            endIndex += charCount;
        }
        return endIndex == chars.length() ? content : chars.subSequence(0, endIndex).toString();
    }

    protected static int utf8Bytes(int codePoint) throws Exception {
        if (codePoint <= 0x7F) {
            return 1;
        }
        if (codePoint <= 0x7FF) {
            return 2;
        }
        if (codePoint <= 0xFFFF) {
            return 3;
        }
        return 4;
    }
}
