package ai.open.right.utils;

public class BytesUtils {

    public static int utf8Bytes(CharSequence sequence) throws Exception {
        int count = 0;
        for (int i = 0, len = sequence.length(); i < len; i++) {
            char ch = sequence.charAt(i);
            if (ch <= 0x7F) {
                count++;
            } else if (ch <= 0x7FF) {
                count += 2;
            } else if (Character.isHighSurrogate(ch)) {
                count += 4;
                i++;
            } else {
                count += 3;
            }
        }
        return count;
    }
}
