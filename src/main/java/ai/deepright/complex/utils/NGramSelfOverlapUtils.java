package ai.deepright.complex.utils;

import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

public class NGramSelfOverlapUtils {

    private static final Object PRESENT = new Object();

    public static int calculateAutoN(String input) throws Exception {
        if (StringUtils.isBlank(input)) return 2;
        int len = input.length();
        long chineseChars = input.chars().filter(c -> c >= 0x4E00 && c <= 0x9FA5).count();
        double chineseRatio = (double) chineseChars / len;
        if (chineseRatio > 0.5) {
            // 中文为主：n 不宜过大，2 或 3 效果最好
            return len > 100 ? 3 : 2;
        } else {
            // 英文/代码为主：
            // 短文本取 4 (识别单词片段)，长文本取 6-8 (识别短语/代码行)
            if (len < 64) return 3;
            if (len < 256) return 4;
            return 6;
        }
    }

    public static double score(String input, int n) throws Exception {
        if (input == null) return 0.0;
        return NGramSelfOverlapUtils.score(input, 0, input.length(), n);
    }

    public static double score(String input, int start, int end, int n) throws Exception {
        if (input == null || n <= 0) return 0.0;
        if (start < 0 || end < start || end > input.length()) {
            throw new IndexOutOfBoundsException("Invalid range: [" + start + ", " + end + ')');
        }
        if (end - start < n) return 0.0;
        Map<GramKey, Object> ngrams = new HashMap<>();
        GramProbe probe = new GramProbe(input);
        int totalGrams = 0;
        for (int i = start; i <= end - n; i++) {
            int gramEnd = i + n;
            if (isBlankRange(input, i, gramEnd)) continue;
            probe.reset(i, gramEnd);
            if (ngrams.get(probe) == null) {
                ngrams.put(new GramKey(input, i, gramEnd, probe.hashCode()), PRESENT);
            }
            totalGrams++;
        }
        if (totalGrams == 0) return 0.0;
        // 重复率 = (总数 - 唯一数) / 总数
        return (double) (totalGrams - ngrams.size()) / totalGrams;
    }

    public static double score(String input) throws Exception {
        int n = calculateAutoN(input);
        return NGramSelfOverlapUtils.score(input, n);
    }

    private static boolean isBlankRange(String input, int start, int end) {
        for (int i = start; i < end; i++) {
            if (input.charAt(i) > ' ') {
                return false;
            }
        }
        return true;
    }

    private static int hashRange(String input, int start, int end) {
        int hash = 1;
        for (int i = start; i < end; i++) {
            hash = 31 * hash + input.charAt(i);
        }
        return hash;
    }

    private static boolean equalsRange(String input, int start, int end, Object other) {
        if (other instanceof GramKey key) {
            return NGramSelfOverlapUtils.sameRange(input, start, end, key.input, key.start, key.end);
        }
        if (other instanceof GramProbe probe) {
            return NGramSelfOverlapUtils.sameRange(input, start, end, probe.input, probe.start, probe.end);
        }
        return false;
    }

    private static boolean sameRange(String left, int leftStart, int leftEnd, String right, int rightStart, int rightEnd) {
        int len = leftEnd - leftStart;
        if (len != rightEnd - rightStart) {
            return false;
        }
        for (int i = 0; i < len; i++) {
            if (left.charAt(leftStart + i) != right.charAt(rightStart + i)) {
                return false;
            }
        }
        return true;
    }

    protected static final class GramKey {

        protected final String input;

        protected final int start;

        protected final int end;

        protected final int hash;

        protected GramKey(String input, int start, int end, int hash) {
            this.input = input;
            this.start = start;
            this.hash = hash;
            this.end = end;
        }

        @Override
        public int hashCode() {
            return hash;
        }

        @Override
        public boolean equals(Object other) {
            return equalsRange(input, start, end, other);
        }
    }

    protected static final class GramProbe {

        private final String input;
        private int start;
        private int end;
        private int hash;

        protected GramProbe(String input) {
            this.input = input;
        }

        protected void reset(int start, int end) {
            this.start = start;
            this.end = end;
            this.hash = hashRange(input, start, end);
        }

        @Override
        public int hashCode() {
            return hash;
        }

        @Override
        public boolean equals(Object other) {
            return equalsRange(input, start, end, other);
        }
    }
}
