package ai.deepright.complex.utils;

import lombok.extern.slf4j.Slf4j;

import java.util.Objects;

@Slf4j
public class ShannonEntropyUtils {

    protected static final int CHARACTER_SPACE = Character.MAX_VALUE + 1;

    protected static final ThreadLocal<int[]> FREQ_BUFFER = ThreadLocal.withInitial(() -> new int[CHARACTER_SPACE]);

    public static double score(String input) throws Exception {
        return ShannonEntropyUtils.score(input, 0, input.length());
    }

    public static double score(String input, int start, int end) throws Exception {
        Objects.requireNonNull(input, "input");
        if (start < 0 || end < start || end > input.length()) {
            throw new IndexOutOfBoundsException("Invalid range: [" + start + ", " + end + ')');
        }
        int len = end - start;
        if (len <= 0) {
            return 0.0;
        }
        int[] freq = FREQ_BUFFER.get();
        int[] touched = new int[Math.min(len, CHARACTER_SPACE)];
        int uniqueCount = 0;
        for (int i = start; i < end; i++) {
            char current = input.charAt(i);
            if (freq[current] == 0) {
                touched[uniqueCount++] = current;
            }
            freq[current]++;
        }
        double h = 0.0;
        double total = len;
        for (int i = 0; i < uniqueCount; i++) {
            int count = freq[touched[i]];
            double p = count / total;
            h -= p * (Math.log(p) / Math.log(2));
            freq[touched[i]] = 0;
        }
        // > 7 加密/压缩/二进制
        // 6-7 base64 高压缩
        // > 5.5 (极高熵)：接近或等于随机噪声，包括加密后的数据流（AES等）、已经压缩过的二进制包、或者是真正的随机数序列
        // > 4.0 (高熵)：优秀的文学作品、诗歌、多语言混合文本。由于用词考究、意象跳跃，统计学上的重复性极低
        // > 2.5 优秀的文学作品、诗歌、多语言混合文本。由于用词考究、意象跳跃，统计学上的重复性极低
        // 日志、代码、配置文件
        return h;
    }
}
